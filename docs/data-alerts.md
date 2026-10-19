# Entries to check

The data-quality card of the dashboard lists entries that look mistyped. Nothing is stored: the list is computed each time the dashboard loads, from the vehicle's own records, so correcting or deleting an entry removes its line. There is nothing to dismiss.

| Alert | Reported when |
| --- | --- |
| Expense far above the usual | A toll, parking fee or other travel expense costs more than 5 times the median of the other expenses of the same category. |
| Energy price outside the usual range | The price per kWh of a charge, or per litre of a fill-up of one fuel type, is more than 50 % above the highest or more than 50 % below the lowest price of the vehicle's other entries. |

Rules shared by both alerts:

- At least 5 other records are needed to compare with; with fewer, nothing is reported.
- Records of the last 12 months are compared; only an entry dated in the last 60 days is reported.
- At most one line per alert, the entry that deviates the most.
- Amounts are in the vehicle's currency. Charges entered in another currency, free entries and expenses that belong to a trip group are left out.
- Charges are checked through their price, not their amount, so a mistyped charge cost appears as a price alert.

The odometer gaps and inconsistencies have their own entry in the "Complete" list of the same card.
