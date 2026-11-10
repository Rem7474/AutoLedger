import type { Component } from 'vue'
import { defineIcon } from './defineIcon'
import { Activity, Car, Disc, LayoutDashboard, Navigation, Receipt, Scale, Wrench } from './index'
import gAlbumFill from '@material-symbols/svg-400/rounded/album-fill.svg?raw'
import gBalanceFill from '@material-symbols/svg-400/rounded/balance-fill.svg?raw'
import gBuildFill from '@material-symbols/svg-400/rounded/build-fill.svg?raw'
import gDashboardFill from '@material-symbols/svg-400/rounded/dashboard-fill.svg?raw'
import gDirectionsCarFill from '@material-symbols/svg-400/rounded/directions_car-fill.svg?raw'
import gMonitoringFill from '@material-symbols/svg-400/rounded/monitoring-fill.svg?raw'
import gNavigationFill from '@material-symbols/svg-400/rounded/navigation-fill.svg?raw'
import gReceiptLongFill from '@material-symbols/svg-400/rounded/receipt_long-fill.svg?raw'

// The filled version of the icons of the destinations: the bar of a phone draws the current one in it, so the active
// place reads from the shape and not from the colour alone. An icon with no entry is drawn the same way when active.
const FILLED = new Map<Component, Component>([
  [Activity, defineIcon('monitoring-fill', gMonitoringFill)],
  [Car, defineIcon('directions_car-fill', gDirectionsCarFill)],
  [Disc, defineIcon('album-fill', gAlbumFill)],
  [LayoutDashboard, defineIcon('dashboard-fill', gDashboardFill)],
  [Navigation, defineIcon('navigation-fill', gNavigationFill)],
  [Receipt, defineIcon('receipt_long-fill', gReceiptLongFill)],
  [Scale, defineIcon('balance-fill', gBalanceFill)],
  [Wrench, defineIcon('build-fill', gBuildFill)],
])

export function filledIcon(icon: Component): Component {
  return FILLED.get(icon) ?? icon
}
