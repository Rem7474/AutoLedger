# Vehicle integrations for Home Assistant

The [AutoLedger module](https://github.com/Rem7474/autoledger-homeassistant) reads the entities you choose, so it works with any Home Assistant integration that exposes them: the odometer, the battery level and the charging state. Charging sessions come from your wallbox or energy meter (see the [wallbox recipes](wallbox-recipes.md)).

| Integration | Brands | Odometer | Battery and charging | Notes |
| :--- | :--- | :--- | :--- | :--- |
| [Tesla Fleet](https://www.home-assistant.io/integrations/tesla_fleet/) | Tesla | Yes, entity disabled by default | Battery, energy added | Official. No trip history. Access to the Fleet API is billed by Tesla. For the full history use the [TeslaMate sync](../README.md#-how-data-gets-in). |
| [Stellantis Vehicles](https://github.com/andreadegiovine/homeassistant-stellantis-vehicles) | Peugeot, Citroën, DS, Opel, Vauxhall | Yes (mileage) | Battery, charging state, last charge | Community, through HACS. Needs the account of the brand's mobile app; waking the car often can drain the 12 V battery. |
| [Kia Uvo / Hyundai Bluelink](https://github.com/Hyundai-Kia-Connect/kia_uvo) | Kia, Hyundai, Genesis | Yes | Battery, range | Community, through HACS. |
| [Renault](https://www.home-assistant.io/integrations/renault/) | Renault | Yes | Battery, range, charging rate | Official. |
| [Volvo](https://www.home-assistant.io/integrations/volvo/) | Volvo | Yes | Battery, range | Official. Also exposes the trip meters. |
| OBD-II ([Torque Pro, WiCAN](https://community.home-assistant.io/t/updated-torque-obd-ii-integration-send-your-cars-data-to-home-assistant/1018554)) | Any brand with an OBD-II port | Yes | Depends on the vehicle | Local hardware or phone app, no manufacturer cloud. |

These integrations are maintained by their own authors and are not affiliated with AutoLedger. The entities that appear depend on the model, the model year and the region: check the documentation of the integration for your vehicle.
