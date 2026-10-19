# Tolls

Tolls are recorded in two ways, and only one of them depends on the country.

| | Available | What it does |
| --- | --- | --- |
| Manual tolls and parking | Everywhere | Add a toll or a parking cost to a drive or a trip, or from the expenses page. It counts in the running cost like any other expense. |
| Automatic detection | Where the toll detection is active | Matches the GPS trace of a TeslaMate drive against French motorway stations, estimates the fare and records it as an automatic toll. It also lists the motorway drives still to qualify (a toll entered, or "no toll"). |

The reference data of the automatic detection is [OpenTollData](https://github.com/louis2038/OpenTollData) and covers French motorways only. Its fares are in euros, so the automatic toll is only recorded for a vehicle whose currency is the euro: for any other vehicle, the drive is skipped and the toll is entered by hand, in its own currency or with an exchange rate.

## Toll detection setting

Each vehicle has a **Toll detection (France)** setting in its form:

| Mode | Effect |
| --- | --- |
| Automatic (default) | Active when the vehicle shows signs of driving in France: a GPS detection that found a toll station, an automatic toll already recorded, a drive whose address names France, or an owner using the French interface. |
| Always on | Active whatever the vehicle did so far, for a first trip to France for instance. |
| Off | Never active, even where the signs above exist. |

The server decides: the vehicle returns `toll_detection` (the mode) and `toll_detection_active` (the result).

When the detection is not active:

- the "To qualify" filter and its counter, the toll source filter, the "Auto toll" bulk action, the qualification buttons of the drive cards and the detection button of the cost modal are hidden;
- the detection and bulk toll endpoints answer `409` with the code `toll.detection_disabled`;
- the motorway drives are not counted in the completeness score of the dashboard;
- manual tolls, the "With toll" filter and the toll line of the cost breakdown stay available, and the tolls already detected are kept.

## History

The Tolls tab opens on the history of what was paid for tolls, parking and other travel expenses: the total and the monthly average of a period (1, 3, 6 or 12 months, the whole history, or a range of days), the 12-month average, and a chart of the amount paid per month with the 12-month average drawn over it. A range of days counts the months it covers entirely. The Maintenance and Fixed costs tabs show the same history for their own expenses; a recurring expense counts once for each month it was due.
