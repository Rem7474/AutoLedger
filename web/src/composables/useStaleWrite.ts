import { useConfirm } from '@/composables/useConfirm'
import { t } from '@/i18n'
import { isStaleWrite } from '@/services/apiError'

/**
 * What an editor does when its save is refused because the item was changed since it was opened: it says so and the
 * caller loads the item again. Resolves true when the error was that one (and has been explained), false for any other.
 */
export function useStaleWrite() {
  const { showAlert } = useConfirm()

  async function explainStaleWrite(err: unknown): Promise<boolean> {
    if (!isStaleWrite(err)) return false
    await showAlert(t('shell.stale.message'), t('shell.stale.title'), 'warning')
    return true
  }

  return { explainStaleWrite }
}
