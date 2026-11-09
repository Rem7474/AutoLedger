// The theme of a cost is one hue, the same for its icon, chip, accent bar, figure and chart series. The colours are
// theme tokens of `assets/main.css` (`--color-theme-*`, used as `text-theme-energy`, `bg-theme-energy/10`); the hex
// values here are the same colours for the canvas of a chart, which cannot read a CSS variable (`themes.test.ts`
// compares the two). Six hues stay apart on the dark surfaces, in normal vision and under colour blindness; the themes with a lesser weight (subscriptions, taxes, accessories, other) are neutral, and
// every chip and series has its label, so a colour is never the only information.

export const THEMES = ['energy', 'tolls', 'tires', 'maintenance', 'repairs', 'insurance', 'financing', 'subscriptions', 'taxes', 'accessories', 'depreciation', 'other'] as const
export type Theme = (typeof THEMES)[number]

/** The hex of each theme, for charts. */
export const THEME_HEX: Record<Theme, string> = {
  energy: '#00bcff',
  tolls: '#ff6900',
  tires: '#53eafd',
  maintenance: '#fb64b6',
  repairs: '#fb64b6',
  insurance: '#bbf451',
  financing: '#7c86ff',
  subscriptions: '#90a1b9',
  taxes: '#cad5e2',
  accessories: '#90a1b9',
  depreciation: '#62748e',
  other: '#90a1b9',
}

export interface ThemeStyle {
  /** Classes of a chip: tinted fill, text and border. */
  badge: string
  /** Classes of the round tonal glyph at the start of a row: tinted fill and icon colour. */
  tint: string
  /** Class of an icon or a figure of the theme. */
  text: string
}

// Written in full: Tailwind finds a class only where its name is spelled out
export const THEME_STYLES: Record<Theme, ThemeStyle> = {
  energy: { badge: 'bg-theme-energy/10 text-theme-energy border-theme-energy/30', tint: 'bg-theme-energy/15 text-theme-energy', text: 'text-theme-energy' },
  tolls: { badge: 'bg-theme-tolls/10 text-theme-tolls border-theme-tolls/30', tint: 'bg-theme-tolls/15 text-theme-tolls', text: 'text-theme-tolls' },
  tires: { badge: 'bg-theme-tires/10 text-theme-tires border-theme-tires/30', tint: 'bg-theme-tires/15 text-theme-tires', text: 'text-theme-tires' },
  maintenance: { badge: 'bg-theme-maintenance/10 text-theme-maintenance border-theme-maintenance/30', tint: 'bg-theme-maintenance/15 text-theme-maintenance', text: 'text-theme-maintenance' },
  repairs: { badge: 'bg-theme-maintenance/10 text-theme-maintenance border-theme-maintenance/30', tint: 'bg-theme-maintenance/15 text-theme-maintenance', text: 'text-theme-maintenance' },
  insurance: { badge: 'bg-theme-insurance/10 text-theme-insurance border-theme-insurance/30', tint: 'bg-theme-insurance/15 text-theme-insurance', text: 'text-theme-insurance' },
  financing: { badge: 'bg-theme-financing/10 text-theme-financing border-theme-financing/30', tint: 'bg-theme-financing/15 text-theme-financing', text: 'text-theme-financing' },
  subscriptions: { badge: 'bg-theme-neutral/10 text-theme-neutral border-theme-neutral/30', tint: 'bg-theme-neutral/15 text-theme-neutral', text: 'text-theme-neutral' },
  taxes: { badge: 'bg-theme-taxes/10 text-theme-taxes border-theme-taxes/30', tint: 'bg-theme-taxes/15 text-theme-taxes', text: 'text-theme-taxes' },
  accessories: { badge: 'bg-theme-neutral/10 text-theme-neutral border-theme-neutral/30', tint: 'bg-theme-neutral/15 text-theme-neutral', text: 'text-theme-neutral' },
  depreciation: { badge: 'bg-theme-neutral/10 text-theme-neutral border-theme-neutral/30', tint: 'bg-theme-neutral/15 text-theme-neutral', text: 'text-theme-neutral' },
  other: { badge: 'bg-theme-neutral/10 text-theme-neutral border-theme-neutral/30', tint: 'bg-theme-neutral/15 text-theme-neutral', text: 'text-theme-neutral' },
}

/** The theme of an expense category (`MAINTENANCE`, `REPAIR`, …); a category with no theme of its own is neutral. */
const EXPENSE_THEMES: Record<string, Theme> = {
  INSURANCE: 'insurance',
  SUBSCRIPTION: 'subscriptions',
  TAX: 'taxes',
  FINANCING: 'financing',
  MAINTENANCE: 'maintenance',
  REPAIR: 'repairs',
  ACCESSORY: 'accessories',
}
export const themeOfExpense = (category: string): Theme => EXPENSE_THEMES[category] ?? 'other'
