# Connecter un assistant (MCP)

AutoLedger peut servir un point d'accès [Model Context Protocol](https://modelcontextprotocol.io), pour qu'un assistant (Claude, Cursor, tout client MCP) réponde à des questions sur le coût de vos véhicules. Il ne fait que lire : aucun outil ne modifie le compte. English: [mcp.md](mcp.md).

## L'activer

Définissez `AUTOLEDGER_MCP_ENABLED=true` et `AUTOLEDGER_BASE_URL` avec l'adresse publique de l'instance (celle qu'utilisera l'assistant, en `https`). Le point d'accès est `<URL de base>/mcp`. Tant que la variable n'est pas définie, le point d'accès et les pages d'autorisation n'existent pas.

Le reverse proxy doit transmettre l'en-tête `Authorization` et l'en-tête `Host` d'origine, comme pour le reste de l'application.

## Connecter un client

Donnez l'adresse du point d'accès au client. Il découvre seul le serveur d'autorisation, s'enregistre et ouvre la page de consentement d'AutoLedger dans le navigateur ; l'approuver crée l'accès (voir [Autoriser un assistant](oauth.fr.md)).

```
claude mcp add --transport http autoledger https://ledger.example/mcp
```

Authentifiez-vous ensuite depuis le client (`/mcp` dans Claude Code). Un client qui demande un jeton au lieu de lancer l'autorisation (un script, une automatisation) peut utiliser un jeton d'API en lecture seule créé dans la page du compte, envoyé en `Authorization: Bearer al_live_...`. Un jeton de portée `ingest` est refusé : il peut envoyer des événements, et un assistant qui lit ne doit pas le détenir.

L'accès figure dans les jetons d'API de la page du compte, et le supprimer à cet endroit coupe l'assistant immédiatement.

## Outils

| Outil | Retourne |
|---|---|
| `list_vehicles` | Les véhicules du compte : `id`, `name`, `make`, `model`, `powertrain`, `currency`, `odometer`. |
| `get_vehicle_costs` | Ce qu'un véhicule a coûté sur ses 12 derniers mois et depuis son achat, par catégorie, avec le coût au km et un score de complétude. Prend un `vehicle_id`. |
| `get_vehicle_data_gaps` | Ce qui manque aux données : trous et anomalies de compteur, charges sans coût, trajets non qualifiés, distance non suivie, assurance ou acquisition manquante. Prend un `vehicle_id`. |

Les outils retournent les mêmes vues que l'[API de lecture](ingestion-api.fr.md#api-de-lecture), avec les mêmes conventions : un montant est une chaîne décimale dans la devise de son véhicule et n'est jamais converti, une distance est en kilomètres, une valeur inconnue est `null`. Les VIN, plaques, adresses, positions, documents et identifiants de trajet ne sont jamais retournés. Un véhicule que le compte ne voit pas est signalé comme introuvable.

## Remarques

- Le point d'accès est sans état : chaque requête porte son propre jeton et rien n'est conservé entre deux requêtes.
- Les requêtes sont limitées à 120 par minute et par adresse.
- Une instance de démonstration en lecture seule (`AUTOLEDGER_DEMO`) refuse les requêtes dont le point d'accès a besoin.
