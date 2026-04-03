import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'

export default function NotFoundPage() {
  const { t } = useTranslation()

  return (
    <div className="flex flex-col items-center justify-center gap-4 py-20">
      <h2 className="text-4xl font-bold text-muted-foreground">404</h2>
      <p className="text-muted-foreground">{t('error.pageNotFound')}</p>
      <Link
        to="/"
        className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
      >
        {t('error.backToHome')}
      </Link>
    </div>
  )
}
