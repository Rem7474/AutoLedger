# Péages

Les péages s'enregistrent de deux façons, dont une seule dépend du pays.

| | Disponible | Ce que ça fait |
| --- | --- | --- |
| Péages et parkings manuels | Partout | Ajoute un péage ou un parking à un trajet ou à un voyage, ou depuis la page des dépenses. Il compte dans le coût d'usage comme toute autre dépense. |
| Détection automatique | Là où la détection des péages est active | Compare la trace GPS d'un trajet TeslaMate aux gares de péage des autoroutes françaises, estime le tarif et l'enregistre comme péage automatique. Elle liste aussi les trajets autoroutiers restant à qualifier (un péage saisi, ou « pas de péage »). |

Les données de référence de la détection automatique viennent d'[OpenTollData](https://github.com/louis2038/OpenTollData) et ne couvrent que les autoroutes françaises. Ses tarifs sont en euros : le péage automatique n'est donc enregistré que pour un véhicule dont la devise est l'euro. Pour tout autre véhicule, le trajet est ignoré et le péage se saisit à la main, dans sa propre devise ou avec un taux de change.

## Réglage de la détection des péages

Chaque véhicule a un réglage **Détection des péages (France)** dans son formulaire :

| Mode | Effet |
| --- | --- |
| Automatique (défaut) | Active quand le véhicule montre des signes de conduite en France : une détection GPS qui a trouvé une gare de péage, un péage automatique déjà enregistré, un trajet dont l'adresse nomme la France, ou un propriétaire qui utilise l'interface française. |
| Toujours activée | Active quoi qu'ait fait le véhicule jusque-là, par exemple pour un premier voyage en France. |
| Désactivée | Jamais active, même si les signes ci-dessus existent. |

Le serveur décide : le véhicule renvoie `toll_detection` (le mode) et `toll_detection_active` (le résultat).

Quand la détection n'est pas active :

- le filtre « À qualifier » et son compteur, le filtre de source du péage, l'action de lot « Péage auto », les boutons de qualification des cartes de trajet et le bouton de détection de la modale de coût sont masqués ;
- les points d'accès de détection et de péages en lot répondent `409` avec le code `toll.detection_disabled` ;
- les trajets autoroutiers ne comptent pas dans le score de complétude du tableau de bord ;
- les péages manuels, le filtre « Avec péage » et la ligne péages de la ventilation des coûts restent disponibles, et les péages déjà détectés sont conservés.

## Arrêts sur l'autoroute

Un arrêt sur une aire, ou un déplacement vers une borne de recharge, découpe un parcours en plusieurs trajets : l'entrée est sur l'un, la sortie sur un autre. La détection lit ensemble les trajets voisins de celui qu'elle analyse dès qu'un ticket de réseau fermé reste ouvert à sa limite (une entrée sans sortie à la fin, ou une barrière sans entrée au début). Un trajet voisin est joint quand il suit l'arrêt de moins de 3 heures et commence à moins de 300 m de la fin du précédent, dans la limite de 5 trajets.

Le ticket est alors tarifé en entier et son prix est réparti entre les trajets qu'il couvre, selon la distance parcourue sur chacun, de sorte que les parts totalisent le tarif. Chaque trajet enregistre sa part comme péage automatique, et le détail du trajet (sous l'estimation) comme la note de ce péage indiquent le prix complet du ticket et le nombre de trajets qu'il couvre. Quand un trajet voisin n'est pas lisible, le trajet garde une entrée sans sortie identifiée.

Si un péage a été saisi à la main, ou pour un groupe de trajets, sur un autre trajet du même ticket, le péage automatique n'est pas enregistré : le parcours est déjà compté.

## Historique

L'onglet Péages s'ouvre sur l'historique de ce qui a été payé en péages, parkings et autres frais de déplacement : le total et la moyenne mensuelle d'une période (1, 3, 6 ou 12 mois, tout l'historique, ou une plage de jours), la moyenne sur 12 mois, et un graphique des montants payés par mois avec la moyenne sur 12 mois tracée par-dessus. Une plage de jours compte les mois qu'elle couvre entièrement. Les onglets Entretien et Coûts fixes affichent le même historique pour leurs propres dépenses ; une dépense récurrente compte une fois pour chaque mois d'échéance.
