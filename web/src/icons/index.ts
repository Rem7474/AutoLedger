import { defineIcon } from './defineIcon'
import dashboard from '@material-symbols/svg-400/rounded/dashboard.svg?raw'
import navigation from '@material-symbols/svg-400/rounded/navigation.svg?raw'
import bolt from '@material-symbols/svg-400/rounded/bolt.svg?raw'
import receiptLong from '@material-symbols/svg-400/rounded/receipt_long.svg?raw'
import directionsCar from '@material-symbols/svg-400/rounded/directions_car.svg?raw'
import add from '@material-symbols/svg-400/rounded/add.svg?raw'
import person from '@material-symbols/svg-400/rounded/person.svg?raw'
import sync from '@material-symbols/svg-400/rounded/sync.svg?raw'
import speed from '@material-symbols/svg-400/rounded/speed.svg?raw'
import group from '@material-symbols/svg-400/rounded/group.svg?raw'
import balance from '@material-symbols/svg-400/rounded/balance.svg?raw'
import album from '@material-symbols/svg-400/rounded/album.svg?raw'
import assignment from '@material-symbols/svg-400/rounded/assignment.svg?raw'
import wallet from '@material-symbols/svg-400/rounded/account_balance_wallet.svg?raw'
import settings from '@material-symbols/svg-400/rounded/settings.svg?raw'
import build from '@material-symbols/svg-400/rounded/build.svg?raw'
import logout from '@material-symbols/svg-400/rounded/logout.svg?raw'

// The icons of the app, named as the pages and controls that show them. A screen takes its icon from here, so the set is
// one family and changing a glyph is a change of one line.
export const LayoutDashboard = defineIcon('dashboard', dashboard)
export const Navigation = defineIcon('navigation', navigation)
export const Zap = defineIcon('bolt', bolt)
export const Receipt = defineIcon('receipt_long', receiptLong)
export const Car = defineIcon('directions_car', directionsCar)
export const Plus = defineIcon('add', add)
export const UserRound = defineIcon('person', person)
export const RefreshCw = defineIcon('sync', sync)
export const Gauge = defineIcon('speed', speed)
export const Users = defineIcon('group', group)
export const Scale = defineIcon('balance', balance)
export const Disc = defineIcon('album', album)
export const ClipboardList = defineIcon('assignment', assignment)
export const Wallet = defineIcon('account_balance_wallet', wallet)
export const Settings = defineIcon('settings', settings)
export const LogOut = defineIcon('logout', logout)
export const Wrench = defineIcon('build', build)
