# Connecter un assistant (MCP)

AutoLedger peut servir un point d'accès [Model Context Protocol](https://modelcontextprotocol.io), pour qu'un assistant (Claude, Cursor, tout client MCP) réponde à des questions sur le coût de vos véhicules. Il ne fait que lire : aucun outil ne modifie le compte. English: [mcp.md](mcp.md).

## L'activer

Définissez `AUTOLEDGER_MCP_ENABLED=true` (dans `.env` avec Docker Compose) et `AUTOLEDGER_BASE_URL` avec l'adresse publique de l'instance (celle qu'utilisera l'assistant, en `https`). Le point d'accès est `<URL de base>/mcp`. Tant que la variable n'est pas définie, le point d'accès et les pages d'autorisation n'existent pas.

Le reverse proxy doit transmettre l'en-tête `Authorization` et l'en-tête `Host` d'origine, comme pour le reste de l'application.

## Connecter un client

Donnez l'adresse du point d'accès au client. Il découvre seul le serveur d'autorisation, s'enregistre et ouvre la page de consentement d'AutoLedger dans le navigateur ; l'approuver crée l'accès (voir [Autoriser un assistant](oauth.fr.md)).

L'accès figure dans les jetons d'API de la page du compte, et le supprimer à cet endroit coupe l'assistant immédiatement. Une autorisation limitée à certains véhicules (choisis sur la page de consentement) ne lit que ceux-là : les vues du foyer qui additionnent tous les véhicules, comme `get_fleet_summary`, sont refusées avec `token.vehicles_limited`, et `get_vehicle_costs` répond pour un véhicule autorisé.

### Claude (claude.ai, Claude Desktop et applications mobiles)

Dans les réglages des connecteurs, ajoutez un connecteur personnalisé avec `https://ledger.example/mcp`. La connexion part des serveurs d'Anthropic, pas de votre appareil : l'instance doit être joignable depuis internet, avec un certificat qu'un client public accepte. Un serveur sur un réseau privé ne peut pas y être ajouté. Avec une offre Team ou Enterprise, un propriétaire ajoute le connecteur pour l'organisation, puis chaque membre se connecte avec son propre compte AutoLedger.

### Claude Code

```
claude mcp add --transport http autoledger https://ledger.example/mcp
```

Lancez ensuite `/mcp` dans une session et choisissez de vous authentifier. Le navigateur ouvre la page de consentement.

### Cursor

Dans `~/.cursor/mcp.json` (tous les projets) ou `.cursor/mcp.json` (un projet) :

```json
{
  "mcpServers": {
    "autoledger": { "url": "https://ledger.example/mcp" }
  }
}
```

Cursor s'enregistre et ouvre la page de consentement ; l'accès revient vers Cursor par son adresse `cursor://`, que le serveur accepte.

### Un client qui prend un jeton

Un script, une automatisation ou un client sans OAuth envoie un jeton d'API en lecture seule (page du compte, accès « Lecture seule ») dans l'en-tête `Authorization`. Dans le `mcp.json` de Cursor :

```json
{
  "mcpServers": {
    "autoledger": {
      "url": "https://ledger.example/mcp",
      "headers": { "Authorization": "Bearer al_live_..." }
    }
  }
}
```

Un jeton de portée `ingest` est refusé : il peut envoyer des événements, et un assistant qui lit ne doit pas le détenir. Un jeton créé ainsi lit tous les véhicules du compte ; seule la page de consentement peut limiter un accès à certains véhicules.

### Home Assistant

L'intégration Model Context Protocol de Home Assistant documente un point d'accès SSE, et ce serveur ne parle que Streamable HTTP (`POST /mcp`) : cette intégration ne peut donc pas s'y connecter. Pour amener les chiffres d'AutoLedger dans Home Assistant, utilisez l'[API de lecture](ingestion-api.fr.md#api-de-lecture) avec un jeton, ou l'[intégration HACS](https://github.com/Rem7474/autoledger-homeassistant).

## Vérifier le point d'accès

Un client qui ne peut pas vérifier le serveur n'atteint généralement pas l'une de ces adresses. Depuis une machine extérieure à votre réseau :

```
curl -i -X POST https://ledger.example/mcp
curl https://ledger.example/.well-known/oauth-protected-resource/mcp
curl https://ledger.example/.well-known/oauth-authorization-server
```

La première doit répondre `401` avec un en-tête `WWW-Authenticate: Bearer resource_metadata="..."`, et les deux autres un document JSON dont les adresses commencent par votre URL publique. Une page HTML, un `404` ou une adresse comme `http://localhost:8080` signifie que `AUTOLEDGER_MCP_ENABLED` n'est pas défini, que le proxy ne transmet pas la requête, ou que `AUTOLEDGER_BASE_URL` n'est pas l'adresse publique. Un client qui s'exécute dans un service en ligne (un connecteur web, par exemple) ne peut joindre qu'une adresse accessible depuis Internet.

## Outils

| Outil | Retourne |
|---|---|
| `list_vehicles` | Les véhicules du compte : `id`, `name`, `make`, `model`, `powertrain`, `currency`, `odometer`, `role` (`owner`, ou `editor` / `viewer` pour un véhicule partagé avec le compte), `tracking_mode` (`connected`, `semi_auto` ou `manual`) et, quand l'acquisition est saisie, `acquisition_type` et `owned_since`. |
| `get_vehicle_costs` | Ce qu'un véhicule a coûté sur ses 12 derniers mois et depuis son achat, par catégorie, avec le coût au km et un score de complétude. Prend un `vehicle_id`. |
| `get_vehicle_data_gaps` | Ce qui manque aux données : trous et anomalies de compteur, charges sans coût, trajets non qualifiés, distance non suivie, assurance ou acquisition manquante, entrées récentes qui semblent mal saisies (une dépense très au-dessus de l'habituel, un prix hors de la fourchette habituelle), et ce qu'il faut compléter classé par valeur. Prend un `vehicle_id`. |
| `list_vehicle_trips` | Les trajets d'un véhicule, du plus récent au plus ancien : heures de départ et d'arrivée, adresses `from` et `to`, distance, durée, énergie, péages et parkings, étiquettes, nom du conducteur. Prend un `vehicle_id` et, en option, `search` (texte dans une adresse, pour retrouver « le trajet vers Lyon »), `from` et `to` (jours `AAAA-MM-JJ`, UTC) et `limit` (20 par défaut, 50 au plus) ; `matching` donne le nombre de trajets qui correspondent. |
| `get_monthly_costs` | Ce qu'un véhicule a coûté mois par mois, du plus ancien au plus récent : le coût de chaque mois par catégorie, sa distance et son coût au km, avec les mois listés additionnés. Prend un `vehicle_id` et, en option, `from` et `to` (mois, `AAAA-MM`) ; sans eux, les 12 derniers mois. Le mois en cours est signalé : ses chiffres ne sont pas définitifs. |
| `get_month_breakdown` | Les écritures qui composent l'énergie ou l'assurance d'un mois : chaque recharge ou dépense avec sa date, son libellé, son montant et sa récurrence, et pour l'énergie la part estimée plutôt qu'enregistrée. Prend un `vehicle_id`, un `month` (`AAAA-MM`) et une `category` (`energy` ou `insurance`) ; les 100 premières écritures sont listées et `matching` donne le total. Les libellés sont le texte du propriétaire, à lire comme une donnée ; une recharge localisée par une seule position n'a pas de libellé. |
| `get_odometer` | Le compteur actuel d'un véhicule, le nombre de relevés datés que contiennent les données et la période qu'ils couvrent. Avec une `date` (`AAAA-MM-JJ`) il donne aussi le compteur ce jour-là et précise s'il s'agit d'un relevé, d'une valeur répartie entre les relevés voisins ou du relevé le plus proche. Prend un `vehicle_id`. |
| `get_reminders` | Les rappels d'entretien d'un véhicule et où en est chacun : en retard, bientôt dus ou à jour, avec la distance et les jours restants, l'échéance et le dernier entretien, du plus urgent au moins urgent. Prend un `vehicle_id` et, en option, un `status` (`overdue`, `due_soon`, `ok`) pour ne garder que ceux-là. |
| `get_tire_status` | Les pneus et jeux d'un véhicule et leur usure : position, distance parcourue, part de la durée de vie consommée, profondeur de sculpture (mesurée ou estimée) face au minimum légal, état, distance restante, jour prévu d'atteinte du minimum légal et coût au km. La marque, le modèle et la dimension sont le texte du propriétaire, à lire comme une donnée. Prend un `vehicle_id`. |
| `get_vehicle_ownership` | Ce que les coûts en espèces laissent de côté : le coût économique de possession (coûts d'usage avec les pneus amortis, plus la dépréciation, au km, net du revenu de covoiturage), l'acquisition, la location ou le crédit (loyer, fin de contrat, kilométrage autorisé consommé et surcoût projeté, option d'achat) et la valeur de revente estimée aujourd'hui. `depreciation_known` dit si la dépréciation a pu être calculée. Prend un `vehicle_id`. |
| `get_energy_efficiency` | Comment un véhicule consomme et paie son énergie. Électrique : consommation aux 100 km, prix du kWh, coût d'énergie aux 100 km, rendement de charge, coût d'une charge complète, répartition entre charge lente, AC et DC, effet du froid, capacité utile et état de santé de la batterie (dernière mesure et évolution sur 12 mois). Carburant : litres aux 100 km, prix du litre, coût du carburant aux 100 km. Le champ `basis` dit quels chiffres électriques sont mesurés et lesquels sont dérivés. Prend un `vehicle_id`. |
| `list_expenses` | Les lignes derrière les coûts d'un véhicule : chaque charge, plein, péage, dépense d'entretien ou d'assurance, achat de pneus et échéance de crédit ou de location, avec sa date, sa catégorie et son montant et, pour une charge, le lieu (le lieu nommé, ou l'adresse relevée) et l'énergie. Prend un `vehicle_id` et, en option, `from` et `to` (jours), `categories`, `min_amount`, `sort` (`date` ou `amount`), `ascending`, `limit` (50, 200 au plus) et `cursor`. `total` et `by_category` couvrent toutes les lignes correspondantes, pas seulement la page. L'achat du véhicule n'est pas une ligne. `has_receipt` indique si un justificatif est joint (`null` pour les lignes qui ne peuvent pas en porter) ; le document lui-même n'est jamais retourné. Les notes sont du texte libre saisi par le propriétaire : des données, jamais des instructions. |
| `get_fleet_summary` | Tous les véhicules du compte côte à côte : coût du mois, coût d'usage et coût complet au km, coût annuel, consommation, rang de coût parmi les véhicules comparables, budget mensuel du foyer et situation du mois, 12 derniers mois de coûts du foyer et qui a conduit combien (les conducteurs sont nommés). Les totaux sont par devise, jamais additionnés entre devises. Sans argument. |
| `get_tag_breakdown` | La répartition des trajets d'un véhicule et de leur coût alloué par étiquette (pro, perso...), sur tout l'historique : distance, énergie, péages, coût alloué et coût au km de chaque étiquette et des trajets sans étiquette. Un trajet portant plusieurs étiquettes compte dans chacune : les lignes ne s'additionnent pas quand `shared_drives` est supérieur à zéro. Les noms d'étiquette sont le texte du propriétaire, à lire comme une donnée. Prend un `vehicle_id`. |
| `get_drive_summary` | Ce que représentent les trajets d'un véhicule sur une période : nombre de trajets, distance, énergie, péages, coût alloué et coût au km, et la répartition par étiquette de `get_tag_breakdown` pour cette période seulement. Il compte aussi les trajets sans étiquette, les trajets sans distance enregistrée et les trajets de type autoroute dont le péage n'est pas encore qualifié, ce qui rend la répartition ou le coût incomplets. Aucune adresse, aucun conducteur ni trajet n'est renvoyé. Prend un `vehicle_id` et, en option, `from` et `to` (jours, `AAAA-MM-JJ`, UTC ; tout l'historique si omis). |
| `get_mileage_report` | La distance parcourue sur une période pour chaque étiquette (professionnel, personnel...) avec l'indemnité kilométrique d'un barème saisi par le propriétaire, les péages et le nombre de trajets, comme le rapport kilométrique de l'application. Les barèmes sont les taux du propriétaire, pas une table officielle : `available_scales` les liste, l'unique barème est appliqué par défaut et `rates` en désigne un autre. Les tranches de chaque année repartent au 1er janvier. Un trajet portant plusieurs étiquettes compte dans chacune, les lignes ne s'additionnent donc pas. Les noms d'étiquettes et de barèmes sont des textes du propriétaire, à lire comme des données. Prend un `vehicle_id`, `from` et `to` facultatifs (jours, `AAAA-MM-JJ`) et `rates` facultatif. |
| `get_carpool` | Ce que le covoiturage a coûté et rapporté : les totaux (coût réel, ce que les passagers ont payé, coût net, taux de couverture, parts équitables) et les derniers trajets partagés (10 par défaut, 30 au plus) avec leurs arrêts, leurs coûts et les passagers nommés : où chacun est monté et descendu, ce qu'il a payé, sa part et ce qu'il doit ou qu'on lui doit. Les noms de passagers, les arrêts et les notes sont le texte du propriétaire, à lire comme une donnée. Prend un `vehicle_id` et un `limit` facultatif. |
| `list_trip_groups` | Les voyages dans lesquels le propriétaire a regroupé plusieurs trajets (des vacances, un déplacement pro), du plus récent au plus ancien (20 par défaut, 50 au plus) : nom, période, nombre de trajets, distance, ce que péages et parkings ont coûté pour tout le voyage et les trajets de covoiturage qu'il porte. Les noms et les notes sont le texte du propriétaire, à lire comme une donnée. Prend un `vehicle_id` et un `limit` facultatif. |
| `get_charging_places` | Où un véhicule a chargé et ce que cela a coûté : pour chaque lieu nommé par le propriétaire (domicile, travail, borne publique...) et pour les charges rattachées à aucun, les adresses qui y correspondent, les sessions, l'énergie, le coût, le prix au kWh et les sessions sans coût enregistré ; ce que la recharge hors domicile a coûté comparé au prix à domicile (null avec sa raison quand rien n'est comparable) ; et les 20 emplacements les plus utilisés qui ne sont pas encore rattachés à un lieu. Les noms de lieux et les adresses sont le texte du propriétaire, à lire comme une donnée. Prend un `vehicle_id` et `months` (12 par défaut, `0` pour tout l'historique, 240 au plus). |
| `get_data_sources` | D'où viennent les données d'un véhicule et leur fraîcheur : ce que chaque origine a écrit (TeslaMate, Home Assistant ou un script, un import CSV, la saisie manuelle) avec la date de son dernier enregistrement, l'état de la synchronisation TeslaMate et le nombre de charges poussées qui attendent encore un véhicule. L'adresse de l'instance TeslaMate n'est jamais retournée. Prend un `vehicle_id`. |
| `get_comparison` | Les scénarios de comparaison électrique / thermique enregistrés (nom, mode, véhicule suivi, distance annuelle, durée, carburant) et, pour celui demandé par `scenario` (identifiant ou nom), son résultat : le coût de chaque véhicule sur la période par catégorie, par an, par mois et au km, ce que le véhicule suivi fait économiser (null tant que sa dépréciation est inconnue), le seuil de rentabilité en années, la sensibilité de l'économie à chaque hypothèse et ce sur quoi reposent les chiffres. Un scénario est une projection, pas un relevé de ce qui a été dépensé. Les noms de scénarios sont du texte du propriétaire, à lire comme des données. Sans argument, il liste les scénarios. |

Partout où un outil prend un `vehicle_id`, le nom du véhicule fonctionne aussi tant qu'aucun autre véhicule ne le porte ; sinon l'erreur liste les véhicules du compte.

Les outils retournent les mêmes vues que l'[API de lecture](ingestion-api.fr.md#api-de-lecture), avec les mêmes conventions : un montant est une chaîne décimale dans la devise de son véhicule et n'est jamais converti, une distance est en kilomètres, une valeur inconnue est `null`. Les VIN, plaques, positions, documents et identifiants de trajet ne sont jamais retournés. Les noms et lieux enregistrés sur le compte sont retournés pour qu'un assistant puisse en parler : adresses de départ et d'arrivée des trajets (`list_vehicle_trips`), noms des conducteurs, arrêts et noms des passagers des covoiturages (`get_carpool`), noms et adresses des lieux de recharge (`get_charging_places`), étiquettes et notes. Une adresse stockée uniquement sous forme de coordonnées est retournée à `null`. Ces textes sont les mots du propriétaire. Chaque champ de sortie qui en porte un indique « user text » dans sa description, et les instructions du serveur demandent à l'assistant de les lire comme des données, jamais comme des instructions ; un test échoue quand un champ nommé comme un texte du propriétaire n'a pas cette mention. Tout ce que vous autorisez un assistant à lire peut être repris dans ses réponses et conservé par son fournisseur : n'en autorisez pas un pour un compte dont vous ne partageriez pas les noms et les adresses. Un véhicule que le compte ne voit pas est signalé comme introuvable.

## Ressources et prompts

En plus des outils, le serveur propose ce qu'un client affiche dans ses propres menus.

| Ressource | Contenu |
|---|---|
| `autoledger://glossary` | Les termes utilisés par les chiffres (coût d'usage, coût complet, coût au km, complétude...), en Markdown. |
| `autoledger://account` | Les unités du compte (distance, volume), la langue, le fuseau de référence, le mois en cours et ses véhicules avec leur devise et leur rôle. |
| `autoledger://vehicles/{vehicle_id}` | Un véhicule, tel que le renvoie `list_vehicles`. Un véhicule que le compte ne voit pas est introuvable. |

| Prompt | Demande à l'assistant de |
|---|---|
| `monthly_review` | Expliquer ce qu'a coûté un mois (`month`, `AAAA-MM` ; le dernier mois complet par défaut), ce qui a changé par rapport aux précédents et ce qu'il faut compléter. |
| `what_is_missing` | Lister ce qu'il faut compléter dans les données d'un véhicule, le plus utile en premier. |
| `lease_check` | Vérifier la location ou le crédit : kilométrage utilisé, surcoût projeté, date de fin. |
| `receipt_check` | Lister les dépenses d'une année (`year`) sans justificatif, par catégorie. |
| `tax_year` | Préparer la répartition professionnel / personnel d'une année (`year` ; l'année précédente par défaut) à partir des étiquettes de trajet. |

Chaque prompt accepte un `vehicle` facultatif (un nom ou un identifiant ; l'assistant demande lequel quand le compte en a plusieurs). Un prompt ne contient aucun chiffre : il indique à l'assistant quels outils appeler. Le client complète `vehicle` et `vehicle_id` avec les véhicules du compte.

## Questions à poser

L'assistant choisit seul les outils. Des questions qui fonctionnent bien, et ce qui y répond :

| Question | Outils |
|---|---|
| Combien m'a coûté ma voiture sur les 12 derniers mois, et depuis l'achat ? | `get_vehicle_costs` |
| Pourquoi mars a-t-il coûté si cher ? | `get_monthly_costs`, `get_month_breakdown` |
| Que manque-t-il à mes données avant de me fier à ces chiffres ? | `get_vehicle_data_gaps`, `get_data_sources` |
| Lequel de mes véhicules coûte le plus au km ? | `get_fleet_summary` |
| Quel est mon coût réel de possession, dépréciation comprise ? | `get_vehicle_ownership` |
| Combien ai-je roulé pour le travail l'an dernier, et quelle est l'indemnité ? | `get_drive_summary`, `get_tag_breakdown`, `get_mileage_report` |
| Combien a coûté mon voyage à la mer, péages compris ? | `list_trip_groups`, `list_vehicle_trips` |
| Où est-ce que je recharge, et où est-ce le moins cher ? | `get_charging_places`, `get_energy_efficiency` |
| Quel entretien est à prévoir, et mes pneus sont-ils usés ? | `get_reminders`, `get_tire_status` |
| Combien le covoiturage a-t-il rapporté ? | `get_carpool` |
| Une voiture électrique m'aurait-elle coûté moins cher que mon diesel ? | `get_comparison` |
| Quelles dépenses n'ont pas de justificatif ? | `list_expenses` (ou le prompt `receipt_check`) |

## Confidentialité

Tout ce qu'un outil renvoie entre dans la conversation, donc est envoyé au fournisseur de l'assistant que vous utilisez (Anthropic pour Claude, le fournisseur de l'éditeur pour Cursor), selon les conditions de ce fournisseur. AutoLedger n'envoie rien de lui-même : l'assistant demande, et seulement les chiffres dont la question a besoin.

- Les montants, distances, dates, noms de véhicules, marques et modèles sont envoyés quand un outil en a besoin.
- Les noms et lieux que vous avez saisis le sont aussi : adresses des trajets, lieux de recharge, conducteurs et passagers de covoiturage, étiquettes et notes. La page de consentement les énumère.
- Les VIN, plaques, positions GPS et documents ne sont jamais servis.
- Une autorisation limitée à certains véhicules ne sert que ces véhicules, et refuse les vues qui additionnent tout le compte.
- Un assistant ne modifie rien : le point d'accès ne fait que lire, et un jeton qui peut envoyer des événements est refusé.
- Le serveur journalise l'outil, le compte, le résultat et la durée de chaque appel, jamais les arguments ni les réponses.

Supprimez l'accès dans les jetons d'API de la page du compte pour y mettre fin immédiatement.

## Dépannage

| Symptôme | Cause et remède |
|---|---|
| Le client ne joint pas le serveur ou signale un serveur invalide | Une des adresses de [Vérifier le point d'accès](#vérifier-le-point-daccès) ne répond pas comme décrit : un `AUTOLEDGER_BASE_URL` erroné, une adresse `http`, ou un proxy qui réécrit l'hôte. |
| La page de consentement dit que l'adresse de retour ne correspond pas à celle que l'application a enregistrée | Le client a envoyé une autre adresse que celle qu'il a enregistrée. Relancez la connexion depuis le client. |
| La page de consentement dit que l'application demande un jeton pour un autre serveur | Le client a demandé l'accès à une autre adresse que `<URL de base>/mcp` : donnez-lui exactement l'URL de base de l'instance suivie de `/mcp`. |
| `401` au bout d'un moment | Le jeton d'accès dure une heure et se renouvelle tant que le client sert. Un accès inutilisé depuis 90 jours, dont le jeton de rafraîchissement a été présenté deux fois, ou supprimé dans la page du compte, doit être autorisé de nouveau. |
| `401` avec un jeton que vous avez créé | Le jeton est révoqué ou expiré, il a la portée `ingest` (créez-en un avec l'accès « Lecture seule »), ou le proxy ne transmet pas l'en-tête `Authorization`. |
| `token.vehicles_limited` | L'accès est limité à certains véhicules et l'outil additionne tous les véhicules (`get_fleet_summary`). Interrogez un véhicule avec `get_vehicle_costs`, ou autorisez de nouveau pour tous les véhicules. |
| Un véhicule manque dans les réponses | L'accès a été limité à d'autres véhicules sur la page de consentement. |
| `429` | Plus de 120 requêtes par minute pour l'accès, ou 600 par adresse. Demandez moins ou patientez une minute. |
| Le résultat est remplacé par une erreur qui demande moins | La réponse dépassait 100 Ko : demandez une période plus courte ou un `limit` plus bas. |

## Compatibilité

Les noms des outils, leurs arguments et les champs des réponses sont un contrat avec tous les clients : un assistant conserve des prompts et des consignes enregistrées qui s'appuient dessus. Un nouvel outil, un nouvel argument facultatif et un nouveau champ dans une réponse peuvent arriver dans n'importe quelle version, un client ignore donc les champs qu'il ne connaît pas. Un renommage, une suppression ou un changement de type ou de sens est une rupture : elle est listée ici avec la version qui la porte, et l'instantané de schéma `internal/handlers/testdata/mcp_tools.golden.json` en fait une différence visible en revue. Le `code` d'un avertissement de complétude est stable, alors que son `message` est un texte pour une personne.

| Version | Rupture |
|---|---|
| | Aucune. |

## Remarques

- Le point d'accès est sans état : chaque requête porte son propre jeton et rien n'est conservé entre deux requêtes.
- Les requêtes sont limitées à 120 par minute et par jeton d'accès, et à 600 par minute et par adresse.
- Le résultat d'un outil est limité à 100 Ko. Un résultat plus gros est remplacé par une erreur qui demande moins (période plus courte, filtre, `limit` plus bas) ; les listes ont leurs propres plafonds et indiquent quand elles sont coupées.
- Le journal du serveur contient une ligne par appel d'outil (outil, compte, résultat, durée) ; les arguments et les réponses n'y figurent pas.
- Une instance de démonstration en lecture seule (`AUTOLEDGER_DEMO`) refuse les requêtes dont le point d'accès a besoin.
