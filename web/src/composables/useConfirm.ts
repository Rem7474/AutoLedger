import { ref } from 'vue'
import { t } from '@/i18n'
import { useToast } from '@/composables/useToast'

export interface ConfirmOptions {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  type?: 'danger' | 'warning' | 'info' | 'success'
  isAlert?: boolean
  /** Label of a second way forward, shown between cancel and confirm (e.g. "Scrap" next to "Delete") */
  alternativeText?: string
}

export type ConfirmChoice = 'confirm' | 'alternative' | 'cancel'

const isOpen = ref(false)
const options = ref<ConfirmOptions>({
  title: '',
  message: '',
  confirmText: '',
  cancelText: '',
  type: 'danger',
  isAlert: false,
  alternativeText: '',
})

let resolvePromise: ((value: ConfirmChoice) => void) | null = null

export function useConfirm() {
  function openDialog(opts: ConfirmOptions | string): Promise<ConfirmChoice> {
    if (typeof opts === 'string') {
      options.value = {
        title: t('shell.confirm.title'),
        message: opts,
        confirmText: t('common.confirm'),
        cancelText: t('common.cancel'),
        type: 'danger',
        isAlert: false,
        alternativeText: '',
      }
    } else {
      options.value = {
        title: opts.title || (opts.type === 'danger' ? t('shell.confirm.deleteTitle') : t('shell.confirm.title')),
        message: opts.message,
        confirmText: opts.confirmText || (opts.type === 'danger' ? t('common.delete') : t('common.confirm')),
        cancelText: opts.cancelText || t('common.cancel'),
        type: opts.type || 'danger',
        isAlert: false,
        alternativeText: opts.alternativeText || '',
      }
    }
    isOpen.value = true

    return new Promise((resolve) => {
      resolvePromise = resolve
    })
  }

  async function showConfirm(opts: ConfirmOptions | string): Promise<boolean> {
    return (await openDialog(opts)) === 'confirm'
  }

  /** A confirmation with a second way forward: resolves to which button was pressed. */
  function showChoice(opts: ConfirmOptions): Promise<ConfirmChoice> {
    return openDialog(opts)
  }

  /** A success is a transient toast; anything else that needs reading blocks behind a dialog. */
  function showAlert(message: string, title?: string, type: 'info' | 'warning' | 'danger' | 'success' = 'info'): Promise<void> {
    if (type === 'success') {
      useToast().showToast(message)
      return Promise.resolve()
    }
    options.value = {
      title: title || (type === 'danger' ? t('shell.confirm.error') : t('shell.confirm.information')),
      message,
      confirmText: t('shell.confirm.gotIt'),
      cancelText: '',
      type: type || 'info',
      isAlert: true,
      alternativeText: '',
    }
    isOpen.value = true

    return new Promise((resolve) => {
      resolvePromise = () => resolve()
    })
  }

  function settle(choice: ConfirmChoice) {
    isOpen.value = false
    if (resolvePromise) {
      resolvePromise(choice)
      resolvePromise = null
    }
  }

  const onConfirm = () => settle('confirm')
  const onAlternative = () => settle('alternative')
  const onCancel = () => settle('cancel')

  return {
    isOpen,
    options,
    showConfirm,
    showChoice,
    showAlert,
    onConfirm,
    onAlternative,
    onCancel,
  }
}
