# Autoriser un assistant (OAuth)

Un assistant, ou toute application qui parle OAuth 2.1, peut demander un accès en lecture seule à un compte sans que l'utilisateur ait à recopier un jeton. L'instance doit l'activer avec `AUTOLEDGER_MCP_ENABLED=true` (désactivé par défaut). L'assistant appelle ensuite le [point d'accès MCP](mcp.fr.md). English: [oauth.md](oauth.md).

## Ce que voit l'utilisateur

L'application envoie le navigateur vers `/oauth/authorize` sur l'instance. Après connexion, l'utilisateur lit ce que l'application demande et l'approuve ou la refuse :

- Elle peut voir les véhicules (nom, marque, modèle, motorisation, devise, compteur), ce que chacun coûte sur les 12 derniers mois et depuis l'achat, et ce qui manque aux données. Elle voit aussi les noms et lieux saisis : adresses des trajets, lieux de recharge, conducteurs et passagers de covoiturage, étiquettes et notes.
- Quand le compte a plusieurs véhicules, l'utilisateur choisit si l'application les voit tous (y compris ceux ajoutés plus tard) ou seulement ceux qu'il coche. Une autorisation limitée affiche ces véhicules dans la liste des jetons d'API, et les autres véhicules n'existent pas pour elle : ils sont absents des listes et refusés par identifiant. Approuver demande au moins un véhicule.
- Elle ne peut rien modifier, ajouter ni supprimer, et ne voit jamais les VIN, plaques, positions GPS ni documents.

Le nom de l'application est donné par l'application elle-même et n'est pas vérifié : n'approuvez qu'une application que vous venez de lancer. La page indique où l'utilisateur est renvoyé.

Approuver crée une autorisation qui apparaît dans la liste des jetons d'API de la page du compte, marquée comme autorisation OAuth. La supprimer à cet endroit met fin à l'accès immédiatement : le jeton d'accès et le jeton de rafraîchissement cessent de fonctionner.

## Ce que reçoit l'application

Un jeton d'accès de la forme `al_live_...` avec la portée `read`, valable une heure, et un jeton de rafraîchissement (`al_refresh_...`) valable 90 jours, prolongé à chaque utilisation. Le jeton d'accès ouvre l'[API de lecture](ingestion-api.fr.md#api-de-lecture) et rien d'autre. Il y a une autorisation par utilisateur et par application : autoriser de nouveau remplace la précédente.

Un jeton de rafraîchissement ne sert qu'une fois. Présenter un jeton déjà utilisé révoque toute l'autorisation, de sorte qu'une copie volée ne peut pas servir discrètement à côté de l'originale.

## Points d'accès

| Point d'accès | Rôle |
|---|---|
| `GET /.well-known/oauth-authorization-server` | Métadonnées du serveur (RFC 8414). |
| `POST /oauth/register` | Enregistrement dynamique de client (RFC 7591). Clients publics uniquement, au plus 1000 clients par instance et 10 enregistrements par heure et par adresse. Un client sans autorisation est supprimé après 30 jours. |
| `/oauth/authorize` | Page de consentement (servie par l'application web). |
| `POST /oauth/token` | Échange un code d'autorisation ou un jeton de rafraîchissement. |
| `POST /oauth/revoke` | Révoque une autorisation à partir d'un jeton (RFC 7009). |

Règles appliquées par le serveur :

- Flux par code d'autorisation avec PKCE, méthode `S256` uniquement. Le code est valable 60 secondes et ne sert qu'une fois.
- Clients publics uniquement (`token_endpoint_auth_method` `none`) ; la seule portée est `read`.
- Une URI de redirection est en `https`, une adresse de boucle locale avec un port quelconque (`http://127.0.0.1:port/...`), ou un schéma privé d'application de bureau (`cursor://...`). Les fragments, jokers et identifiants dans l'URI sont refusés, ainsi que `javascript:`, `data:` et `file:`.
- Si la requête porte un paramètre `resource` (RFC 8707), il doit être l'URL MCP de l'instance, `<URL de base>/mcp`.
- La redirection porte un paramètre `iss` (RFC 9207) avec l'URL de l'instance.

Les points d'accès de métadonnées, d'enregistrement, de jeton et de révocation, ainsi que le point d'accès MCP, répondent à une requête de n'importe quelle origine (`Access-Control-Allow-Origin: *`, sans identifiants), de sorte qu'un client exécuté dans un navigateur peut les utiliser. Ils ne lisent aucun cookie : le credential est le jeton porteur, ou le code d'autorisation avec son vérificateur PKCE. Le reste de l'API continue de refuser les requêtes d'une autre origine.
