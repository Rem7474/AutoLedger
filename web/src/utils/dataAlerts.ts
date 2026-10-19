import { intlLocale, t } from '@/i18n'
import { formatAmount } from '@/currency'

export interface DataAlert {
  kind: 'amount_outlier' | 'price_outlier'
  category: string
  source_id: string
  date: string
  amount?: number
  median?: number
  unit_price?: number
  low?: number
  high?: number
}

export interface DataAlertLine {
  key: string
  /** The page where the entry is corrected. */
  to: string
  text: string
}

const PRICE_PAGES: Record<string, string> = { CHARGE: '/energy?tab=CHARGES', FUEL: '/energy?tab=FUEL' }

// One line per alert: what is odd, in the vehicle's currency, and where to correct it
export function dataAlertLine(alert: DataAlert, currency: string): DataAlertLine {
  const date = new Date(alert.date).toLocaleDateString(intlLocale(), { day: 'numeric', month: 'short', year: 'numeric' })
  if (alert.kind === 'price_outlier') {
    const unit = alert.category === 'FUEL' ? t('dashboard.dataQualityCard.alerts.perLiter') : t('dashboard.dataQualityCard.alerts.perKwh')
    return {
      key: alert.kind,
      to: PRICE_PAGES[alert.category] ?? PRICE_PAGES.CHARGE,
      text: t('dashboard.dataQualityCard.alerts.priceOutlier', {
        date,
        price: `${formatAmount(alert.unit_price ?? 0, currency, 3)}${unit}`,
        low: formatAmount(alert.low ?? 0, currency, 3),
        high: formatAmount(alert.high ?? 0, currency, 3),
      }),
    }
  }
  const median = alert.median ?? 0
  return {
    key: alert.kind,
    to: '/expenses',
    text: t('dashboard.dataQualityCard.alerts.amountOutlier', {
      category: t(`dashboard.dataQualityCard.alerts.categories.${alert.category}`),
      date,
      amount: formatAmount(alert.amount ?? 0, currency),
      median: formatAmount(median, currency),
      times: median > 0 ? Math.round((alert.amount ?? 0) / median) : 0,
    }),
  }
}
