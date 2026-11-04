# Intégrations véhicule pour Home Assistant

Le [module AutoLedger](https://github.com/Rem7474/autoledger-homeassistant) lit les entités que vous choisissez : il fonctionne donc avec toute intégration Home Assistant qui les expose (kilométrage, niveau de batterie, état de charge). Les sessions de charge viennent de votre wallbox ou de votre compteur d'énergie (voir les [recettes wallbox](wallbox-recipes.fr.md)).

| Intégration | Marques | Kilométrage | Batterie et charge | Remarques |
| :--- | :--- | :--- | :--- | :--- |
| [Tesla Fleet](https://www.home-assistant.io/integrations/tesla_fleet/) | Tesla | Oui, entité désactivée par défaut | Batterie, énergie ajoutée | Officielle. Pas d'historique de trajets. L'accès à l'API Fleet est facturé par Tesla. Pour l'historique complet, utilisez la [synchronisation TeslaMate](../README.fr.md). |
| [Stellantis Vehicles](https://github.com/andreadegiovine/homeassistant-stellantis-vehicles) | Peugeot, Citroën, DS, Opel, Vauxhall | Oui (kilométrage) | Batterie, état de charge, dernière charge | Communautaire, via HACS. Nécessite le compte de l'application de la marque ; réveiller souvent la voiture peut décharger la batterie 12 V. |
| [Kia Uvo / Hyundai Bluelink](https://github.com/Hyundai-Kia-Connect/kia_uvo) | Kia, Hyundai, Genesis | Oui | Batterie, autonomie | Communautaire, via HACS. |
| [Renault](https://www.home-assistant.io/integrations/renault/) | Renault | Oui | Batterie, autonomie, puissance de charge | Officielle. |
| [Volvo](https://www.home-assistant.io/integrations/volvo/) | Volvo | Oui | Batterie, autonomie | Officielle. Expose aussi les compteurs de trajet. |
| OBD-II ([Torque Pro, WiCAN](https://community.home-assistant.io/t/updated-torque-obd-ii-integration-send-your-cars-data-to-home-assistant/1018554)) | Toute marque avec une prise OBD-II | Oui | Selon le véhicule | Matériel local ou application mobile, sans cloud constructeur. |

Ces intégrations sont maintenues par leurs auteurs et ne sont pas affiliées à AutoLedger. Les entités disponibles dépendent du modèle, de l'année et de la région : consultez la documentation de l'intégration pour votre véhicule.

## Connecter Home Assistant à AutoLedger

Le module AutoLedger a besoin d'un accès à votre instance, de deux façons :

- **Connexion OAuth.** L'instance l'active avec `AUTOLEDGER_OAUTH_ENABLED=true` (ou `AUTOLEDGER_MCP_ENABLED=true`, qui l'inclut) et `AUTOLEDGER_BASE_URL` réglé sur son adresse publique. Home Assistant vous envoie vers une page d'autorisation où vous choisissez les véhicules qu'il voit et cochez la case qui lui permet d'enregistrer des sessions de charge, des relevés de compteur et des trajets. Supprimer l'accès dans la liste des jetons d'API de la page Compte y met fin. Voir [OAuth](oauth.fr.md).
- **Jeton d'API.** Créez depuis la page Compte un jeton avec l'accès `homeassistant` : il envoie des événements et lit les chiffres, donc un seul jeton sert les deux sens. Un jeton `ingest` envoie seulement et lit les indicateurs des véhicules, un jeton `read` lit seulement. Voir l'[API d'ingestion](ingestion-api.fr.md#accès).
