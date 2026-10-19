# Saisies à vérifier

La carte de qualité des données du tableau de bord liste les saisies qui semblent erronées. Rien n'est enregistré : la liste est recalculée à chaque chargement du tableau de bord à partir des données du véhicule, donc corriger ou supprimer une saisie fait disparaître sa ligne. Il n'y a rien à écarter.

| Alerte | Signalée quand |
| --- | --- |
| Dépense bien au-dessus de l'habituel | Un péage, un stationnement ou un autre frais de déplacement coûte plus de 5 fois la médiane des autres dépenses de la même catégorie. |
| Prix de l'énergie hors de la fourchette habituelle | Le prix au kWh d'une recharge, ou au litre d'un plein d'un même carburant, dépasse de plus de 50 % le plus haut prix des autres saisies du véhicule, ou est inférieur de plus de 50 % au plus bas. |

Règles communes aux deux alertes :

- Il faut au moins 5 autres enregistrements à comparer ; en dessous, rien n'est signalé.
- Les enregistrements des 12 derniers mois servent de référence ; seule une saisie datée des 60 derniers jours est signalée.
- Au plus une ligne par alerte, la saisie qui s'écarte le plus.
- Les montants sont dans la devise du véhicule. Les recharges saisies dans une autre devise, les saisies gratuites et les dépenses rattachées à un voyage sont écartées.
- Les recharges sont contrôlées par leur prix, pas par leur montant : un coût de recharge mal saisi apparaît donc comme une alerte de prix.

Les trous et incohérences du compteur ont leur propre entrée dans la liste « Compléter » de la même carte.
