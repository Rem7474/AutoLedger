# API d'ingestion

Home Assistant, Node-RED, n8n ou un script peuvent envoyer des événements à AutoLedger en HTTP. English: [ingestion-api.md](ingestion-api.md).

## Authentification

Créez un jeton d'API dans la page du compte (`al_live_...`). Il est lié au compte et n'ouvre que les routes `/api/integrations/**`. Envoyez-le en jeton Bearer :

```
Authorization: Bearer al_live_xxxxxxxx
```

### Accès

Un jeton a un accès, choisi à sa création et jamais modifié ensuite (générez un autre jeton pour en changer) :

| Accès | Permet |
|---|---|
| Envoi et lecture (`ingest`) | Envoyer des événements (`POST /api/integrations/homeassistant/event`) et lire les véhicules et leurs indicateurs. C'est ce dont Home Assistant a besoin. L'API de lecture est refusée. |
| Lecture seule (`read`) | Lire les véhicules et leurs indicateurs, et l'[API de lecture](#api-de-lecture). Un envoi est refusé avec `403`. |

Utilisez un jeton en lecture seule pour tout ce qui ne fait que consulter les chiffres (un tableau de bord, un script, un assistant) : un jeton perdu ou détourné ne peut rien écrire dans le compte.

## Point d'entrée

`POST /api/integrations/homeassistant/event` avec un corps JSON.

| Champ | Type | Remarques |
|---|---|---|
| `event_type` | chaîne | `charging_session_end` (par défaut si vide), `odometer`, `drive`, `fuel`. `odometer_update` et `telemetry_update` sont d'anciens noms de `odometer`. |
| `vehicle_id` | chaîne | Obligatoire pour `odometer`, `drive`, `fuel`. Facultatif pour les sessions de charge. Le compte doit pouvoir modifier le véhicule. |
| `event_id` | chaîne | Votre propre identifiant de l'événement, pour l'idempotence (voir plus bas). |
| `timestamp` | RFC 3339 | Utilisé en l'absence de `data.start_time` / `data.end_time`. |
| `distance_unit` | `km` (défaut) ou `mi` | Unité des distances et du compteur de l'événement. Les valeurs sont stockées en km. |
| `data` | objet | Champs du type d'événement, ci-dessous. |

### `charging_session_end`

`data` : `start_time`, `end_time`, `energy_kwh` (ou `energy_added_kwh` ; supérieur à 0, 500 au plus), `cost` (unités monétaires) ou `cost_cents`, `location`, `charger_name`, `odometer_km`, `soc_start`, `soc_end`.

Sans coût, le plan tarifaire du véhicule s'applique s'il existe, sinon le coût reste à compléter. Véhicules électriques uniquement.

Une session sans `vehicle_id` va au véhicule par défaut de la borne domestique, sinon au seul véhicule électrique modifiable du compte, sinon elle attend dans les charges en attente (borne partagée entre plusieurs véhicules) que vous l'affectiez dans l'application.

### `odometer`

`data` : `odometer` (dans `distance_unit`) ou `odometer_km`. Le compteur du véhicule ne fait qu'avancer.

### `drive`

`data` : `start_time` (ou le `timestamp` de l'événement), `end_time` ou `duration_min`, `distance` (dans `distance_unit`, de 0,1 à 3000 km), `start_odometer`, `end_odometer`, `start_address`, `end_address`, `energy_kwh`.

`distance` est facultative. Si elle est absente, c'est la différence de `end_odometer` et `start_odometer` quand les deux sont envoyés. Sinon le trajet est enregistré sans distance (`distance_km` vaut `null` dans l'API, un tiret dans l'application, une cellule vide dans l'export CSV), sans énergie estimée ni consommation ; une `distance` envoyée à 0 est refusée. `start_lat`/`start_lon` et `end_lat`/`end_lon` donnent les positions : elles tiennent lieu de `start_address` / `end_address` quand ceux-ci sont absents (enregistrées sous la forme `lat, lon`). Quand `GEOCODING_ENABLED` est activé sur le serveur, ces positions sont ensuite remplacées en arrière-plan par une adresse trouvée sur OpenStreetMap Nominatim (une adresse modifiée entre-temps est conservée ; en cas d'échec les coordonnées restent).

Sans `energy_kwh`, l'énergie est estimée d'après la consommation moyenne du véhicule et marquée comme estimée.

### `fuel`

Véhicules thermiques uniquement. `data` : `amount` (ou `cost`), `liters`, `price_per_liter`, `fuel_type`, `is_full_tank`, `odometer` ou `odometer_km`. Le montant peut se déduire des litres et du prix au litre. L'heure est le `timestamp` de l'événement (ou `data.start_time`), maintenant à défaut.

## Réponses

| Statut | `status` | Signification |
|---|---|---|
| 201 | `recorded` | Enregistré. Le corps contient `vehicle_id` et l'identifiant du nouvel enregistrement. |
| 201 | `pending_qualification` | Session de charge en attente d'un véhicule (`pending_id`). |
| 200 | `duplicate` | Déjà enregistré ou déjà en attente ; rien n'a été écrit. |
| 200 | (compteur) | Relevé appliqué. |

Les erreurs répondent `{"error": "...", "code": "...", "params": {...}}`. Codes fréquents :

| Code | Cause |
|---|---|
| `request.invalid_body` | Le corps n'est pas du JSON valide. |
| `integration.unknown_event_type` | `event_type` mal orthographié. |
| `integration.invalid_distance_unit` | `distance_unit` n'est ni `km` ni `mi`. |
| `vehicle.not_specified` | `vehicle_id` manquant sur un événement qui l'exige. |
| `telemetry.missing_odometer` | Aucun relevé de compteur valide. |
| `charge.invalid_energy`, `charge.invalid_period`, `charge.electric_only` | Session de charge refusée. |
| `drive.missing_start_time`, `drive.invalid_distance` | Trajet refusé. |
| `fuel.combustion_only`, `fuel.amount_required` | Plein refusé. |

## Idempotence

Renvoyer le même événement est sans danger. Un enregistrement est reconnu comme déjà stocké :

1. par `event_id` (sessions de charge et trajets), quand vous en envoyez un ;
2. sinon par tolérance : une charge débutant à 30 min près avec une énergie à 0,5 kWh près, un trajet débutant à 15 min près avec une distance à 1 km près, un plein à 30 min près pour le même montant (les pleins ne sont reconnus que de cette façon).

Utilisez un `event_id` stable (identifiant de session, clé construite sur l'horodatage) dès que la source en fournit un.

## API de lecture

Un jeton en lecture seule (`read`) ouvre aussi une API de lecture restreinte sous `/api/integrations/v1`, pensée pour les assistants et les rapports. Un jeton `ingest` y est refusé : un jeton qui peut envoyer des événements ne peut pas aussi lire les coûts.

| Route | Renvoie |
|---|---|
| `GET /api/integrations/v1/vehicles` | Les véhicules du compte : `id`, `name`, `make`, `model`, `powertrain`, `currency`, `odometer`, `role`, `tracking_mode` et, quand l'acquisition est saisie, `acquisition_type` et `owned_since`. |
| `GET /api/integrations/v1/vehicles/{vehicleId}` | Un véhicule, mêmes champs. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/costs` | Ce que le véhicule a coûté sur ses 12 derniers mois (`last_12_months`) et depuis son achat (`since_purchase`), par catégorie, avec le coût au km et un résumé `completeness`. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/data-gaps` | Ce qui manque à l'historique : trous et anomalies de compteur (les 20 plus récents sont listés), charges sans coût, trajets non qualifiés, distance non suivie, assurance ou acquisition absentes, le détail de ce qui manque à l'acquisition, `suspicious_entries` (entrées récentes qui semblent mal saisies, une au plus par type), et `missing` : ce qu'il faut compléter, classé comme dans l'application, avec une valeur approximative (`estimated_amount`, `null` quand elle ne peut pas être estimée) et les `score_dimensions` du score de complétude. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/trips` | Les trajets, du plus récent au plus ancien, avec leurs adresses `from` et `to`, distance, durée, énergie, `travel_expenses` (péages, parkings) et étiquettes. Paramètres : `search` (texte dans une adresse), `from` et `to` (jours `AAAA-MM-JJ` en UTC, ou instants RFC 3339), `limit` (20 par défaut, 50 au plus). `matching` compte les trajets qui correspondent et `truncated` indique qu'il en existe plus que ceux listés. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/monthly-costs` | Le coût de chaque mois calendaire, du plus ancien au plus récent, par catégorie, avec la distance et le coût au km du mois, et les mois listés additionnés (`total`). Paramètres : `from` et `to` (mois, `AAAA-MM`, inclus) ; sans eux les 12 derniers mois, au plus 120 mois sinon. Un mois sans enregistrement est listé avec des montants nuls, et le mois en cours porte `in_progress: true`. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/reminders` | Les rappels d'entretien du véhicule, du plus urgent au moins urgent, chacun avec son statut (`overdue`, `due_soon`, `ok`), la distance et les jours restants, son kilométrage et sa date d'échéance et son dernier entretien, plus le nombre de rappels en retard et bientôt dus. Paramètre : `status` (`overdue`, `due_soon` ou `ok`) pour ne garder que ceux-là ; les compteurs l'ignorent. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/ownership` | Le coût économique de possession du véhicule (`economic_cost` : coût complet, dépréciation et si elle est connue, taux au km, revenu de covoiturage), l'acquisition, la location ou le crédit (`contract`, avec le kilométrage autorisé et le surcoût projeté) et la valeur de revente estimée (`resale`, avec la raison quand elle ne peut pas l'être). |
| `GET /api/integrations/v1/vehicles/{vehicleId}/energy-efficiency` | Consommation, prix et coûts de l'énergie aux 100 km, répartition des charges, effet du froid et santé de la batterie pour le côté électrique (`electric`, null si le véhicule ne se recharge pas), et litres aux 100 km et prix du carburant pour le côté carburant (`fuel`, null s'il ne brûle pas de carburant). |
| `GET /api/integrations/v1/vehicles/{vehicleId}/expenses` | Les lignes du registre de coûts du véhicule (charges, carburant, péages, entretien, assurance, pneus, échéances de crédit et de location), la plus récente d'abord, avec le total et les sommes par catégorie de toutes les lignes correspondantes. Paramètres : `from`, `to` (jours, `AAAA-MM-JJ`), `category` (répétable ou séparé par des virgules), `min_amount`, `sort` (`date` ou `amount`), `order` (`asc`), `limit` (50, 200 au plus) et `cursor` (le `next_cursor` de la page précédente). L'achat n'est pas une ligne. |
| `GET /api/integrations/v1/fleet` | Les véhicules du compte côte à côte (coût du mois, coût d'usage et complet au km, coût annuel, consommation, rang de coût parmi les véhicules comparables d'une devise), les totaux du mois par devise, le budget mensuel et son utilisation, les 12 derniers mois de coûts du foyer dans la devise de la flotte, et la part de distance de chaque conducteur nommé. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/tags` | Les trajets d'un véhicule répartis par étiquette sur tout l'historique : distance, énergie, péages, coût alloué et coût au km de chaque étiquette et des trajets sans étiquette. Un trajet portant plusieurs étiquettes compte dans chacune. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/carpool` | Les totaux de covoiturage d'un véhicule et ses derniers trajets partagés (`limit`, 10 par défaut, 30 au plus) avec leurs arrêts, leurs coûts et les passagers nommés. |

Conventions :

- Un montant est `{"amount": "12.34", "currency": "GBP"}` : une chaîne décimale, dans la devise du véhicule. Les montants ne sont jamais convertis d'une devise à l'autre.
- Une distance est `{"value": 4200, "unit": "km"}`. L'unité est toujours le kilomètre.
- Une valeur impossible à connaître (un coût au km sans distance, un score sans rien à évaluer) vaut `null`, jamais `0`.
- `months_covered` indique combien de mois calendaires couvre une période : `last_12_months` d'un véhicule à l'historique plus court en couvre moins de 12.
- Les coûts sont les sommes décaissées, achat du véhicule exclu. Le mois en cours compte les échéances qui tombent plus tard dans le mois.
- Les réponses ne contiennent ni VIN, ni plaque, ni coordonnée, ni document, ni identifiant de trajet. Les trajets sont la seule vue avec des adresses (`from` et `to`) ; une adresse stockée uniquement sous forme de coordonnées vaut `null`.
- Un véhicule que le compte ne peut pas voir répond `404`.

## Exemples

curl :

```bash
curl -X POST https://autoledger.example.com/api/integrations/homeassistant/event \
  -H "Authorization: Bearer al_live_xxxxxxxx" -H "Content-Type: application/json" \
  -d '{"event_type":"charging_session_end","event_id":"wallbox-2026-05-01-1","vehicle_id":"<id>",
       "data":{"start_time":"2026-05-01T22:00:00Z","end_time":"2026-05-02T05:30:00Z","energy_kwh":38.4,"cost":5.76}}'
```

Un trajet en miles :

```bash
curl -X POST https://autoledger.example.com/api/integrations/homeassistant/event \
  -H "Authorization: Bearer al_live_xxxxxxxx" -H "Content-Type: application/json" \
  -d '{"event_type":"drive","vehicle_id":"<id>","distance_unit":"mi",
       "data":{"start_time":"2026-05-01T08:00:00Z","duration_min":25,"distance":12.4,"end_odometer":42310}}'
```

Home Assistant (`rest_command`) :

```yaml
rest_command:
  autoledger_odometer:
    url: "https://autoledger.example.com/api/integrations/homeassistant/event"
    method: POST
    headers:
      Authorization: "Bearer al_live_xxxxxxxx"
      Content-Type: "application/json"
    payload: >
      {"event_type":"odometer","vehicle_id":"<id>","data":{"odometer_km":{{ states('sensor.car_odometer') | float }}}}
```

Node-RED / n8n : un nœud HTTP Request, méthode `POST`, l'URL ci-dessus, l'en-tête `Authorization` et le corps JSON de l'événement.
