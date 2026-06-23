import Alert from '@/components/ui/alert/Alert';

interface ParticipantLockBannerProps {
  /** Имя администратора, удерживающего лок. Если не задано — баннер не рисуется. */
  ownerName?: string;
}

// ParticipantLockBanner показывает предупреждение, когда запись участника уже
// редактирует другой администратор. Рендерится только при наличии чужого лока.
export default function ParticipantLockBanner({ ownerName }: ParticipantLockBannerProps) {
  if (!ownerName) return null;

  return (
    <Alert
      variant="warning"
      title="Запись редактируется"
      message={`Эту запись сейчас редактирует ${ownerName}. Изменения недоступны, пока редактирование не завершится.`}
    />
  );
}
