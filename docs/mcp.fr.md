# Connecter un assistant (MCP)

AutoLedger peut servir un point d'accès [Model Context Protocol](https://modelcontextprotocol.io), pour qu'un assistant (Claude, Cursor, tout client MCP) réponde à des questions sur le coût de vos véhicules. Il ne fait que lire : aucun outil ne modifie le compte. English: [mcp.md](mcp.md).

## L'activer

Définissez `AUTOLEDGER_MCP_ENABLED=true` (dans `.env` avec Docker Compose) et `AUTOLEDGER_BASE_URL` avec l'adresse publique de l'instance (celle qu'utilisera l'assistant, en `https`). Le point d'accès est `<URL de base>/mcp`. Tant que la variable n'est pas définie, le point d'accès et les pages d'autorisation n'existent pas.

Le reverse proxy doit transmettre l'en-tête `Authorization` et l'en-tête `Host` d'origine, comme pour le reste de l'application.

## Connecter un client

Donnez l'adresse du point d'accès au client. Il découvre seul le serveur d'autorisation, s'enregistre et ouvre la page de consentement d'AutoLedger dans le navigateur ; l'approuver crée l'accès (voir [Autoriser un assistant](oauth.fr.md)).

```
claude mcp add --transport http autoledger https://ledger.example/mcp
```

Authentifiez-vous ensuite depuis le client (`/mcp` dans Claude Code). Un client qui demande un jeton au lieu de lancer l'autorisation (un script, une automatisation) peut utiliser un jeton d'API en lecture seule créé dans la page du compte, envoyé en `Authorization: Bearer al_live_...`. Un jeton de portée `ingest` est refusé : il peut envoyer des événements, et un assistant qui lit ne doit pas le détenir.

L'accès figure dans les jetons d'API de la page du compte, et le supprimer à cet endroit coupe l'assistant immédiatement.

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
| `get_vehicle_data_gaps` | Ce qui manque aux données : trous et anomalies de compteur, charges sans coût, trajets non qualifiés, distance non suivie, assurance ou acquisition manquante, et ce qu'il faut compléter classé par valeur. Prend un `vehicle_id`. |
| `list_vehicle_trips` | Les trajets d'un véhicule, du plus récent au plus ancien : heures de départ et d'arrivée, adresses `from` et `to`, distance, durée, énergie, péages et parkings, étiquettes. Prend un `vehicle_id` et, en option, `search` (texte dans une adresse, pour retrouver « le trajet vers Lyon »), `from` et `to` (jours `AAAA-MM-JJ`, UTC) et `limit` (20 par défaut, 50 au plus) ; `matching` donne le nombre de trajets qui correspondent. |
| `get_monthly_costs` | Ce qu'un véhicule a coûté mois par mois, du plus ancien au plus récent : le coût de chaque mois par catégorie, sa distance et son coût au km, avec les mois listés additionnés. Prend un `vehicle_id` et, en option, `from` et `to` (mois, `AAAA-MM`) ; sans eux, les 12 derniers mois. Le mois en cours est signalé : ses chiffres ne sont pas définitifs. |

Partout où un outil prend un `vehicle_id`, le nom du véhicule fonctionne aussi tant qu'aucun autre véhicule ne le porte ; sinon l'erreur liste les véhicules du compte.

Les outils retournent les mêmes vues que l'[API de lecture](ingestion-api.fr.md#api-de-lecture), avec les mêmes conventions : un montant est une chaîne décimale dans la devise de son véhicule et n'est jamais converti, une distance est en kilomètres, une valeur inconnue est `null`. Les VIN, plaques, positions, documents et identifiants de trajet ne sont jamais retournés. Les adresses n'apparaissent qu'avec `list_vehicle_trips` : il retourne les adresses de départ et d'arrivée enregistrées sur le compte, pour qu'un assistant puisse parler d'un trajet par sa destination. Une adresse stockée uniquement sous forme de coordonnées est retournée à `null`. Tout ce que vous autorisez un assistant à lire peut être repris dans ses réponses et conservé par son fournisseur : n'en autorisez pas un pour un compte dont vous ne partageriez pas les adresses de trajet. Un véhicule que le compte ne voit pas est signalé comme introuvable.

## Remarques

- Le point d'accès est sans état : chaque requête porte son propre jeton et rien n'est conservé entre deux requêtes.
- Les requêtes sont limitées à 120 par minute et par jeton d'accès, et à 600 par minute et par adresse.
- Le journal du serveur contient une ligne par appel d'outil (outil, compte, résultat, durée) ; les arguments et les réponses n'y figurent pas.
- Une instance de démonstration en lecture seule (`AUTOLEDGER_DEMO`) refuse les requêtes dont le point d'accès a besoin.
