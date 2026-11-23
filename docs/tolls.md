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

## Stops on a motorway

A stop at a service area, or a move to a charger, splits a journey into several drives: the entry is on one, the exit on another. The detection reads the drives around the one it analyses together as soon as a closed-network ticket stays open at its edge (an entry with no exit at its end, or a gate with no entry at its start). A neighbouring drive joins when it follows the stop by less than 3 hours and begins within 300 m of where the previous one ended, up to 5 drives in all.

The ticket is then priced as a whole and its price is shared between the drives it covers, along the distance travelled on each, so that the shares add up to the fare. Each drive records its own share as an automatic toll, and both the detail of the drive (next to the estimate) and the note of that toll give the full price of the ticket and the number of drives it covers. When a neighbouring drive cannot be read, the drive keeps an entry with no identified exit.

If a toll was entered by hand, or for a trip group, on another drive of the same ticket, the automatic toll is not recorded: the journey is already accounted for.

## Monthly invoice

The tolls entered or estimated trip by trip are an approximation of what the operator bills (subscription discounts, journeys never entered). Once the monthly statement arrives, its total can be recorded for that month, from the Costs page (Tolls filter) or with the `set_toll_invoice` tool of the assistant.

- A month with an invoice counts the invoice total as its tolls in the monthly and global figures (cost per kilometre, monthly costs, totals, fleet). The individual tolls of that month, entered or automatic, no longer add to it. Parking and other travel expenses are not tolls and are unaffected.
- Trips, carpools and trip groups keep the tolls entered on them exactly as before: the invoice never edits them.
- The gap between the invoice and the tolls recorded on trips is shown with it: positive when the invoice holds tolls that no trip carries, negative when the trips record more than was billed.
- A vehicle has one invoice per month, which can be changed or deleted at any time; deleting it brings the individual tolls back into the figures. An invoice of 0 says the month had no toll. The total is in the vehicle's currency.

## History

On the Costs page, the Tolls filter lists what was paid for tolls, parking and other travel expenses over the period chosen at the top, with the total paid and the cost per kilometre; a line opens its record for editing, and swiping it deletes it with a way back. The Maintenance filter does the same for servicing and repairs, and the whole cost adds the figures by month. A recurring expense counts once for each month it was due.
