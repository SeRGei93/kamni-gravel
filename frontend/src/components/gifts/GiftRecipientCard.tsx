import Badge from '@/components/ui/badge/Badge';
import type { ManualGift } from '@/types';
import type { AutomaticGiftRecipient } from '@/utils/giftRecipient';

interface GiftRecipientCardProps {
  manualGift?: ManualGift;
  automaticRecipient?: AutomaticGiftRecipient;
}

export default function GiftRecipientCard({
  manualGift,
  automaticRecipient,
}: GiftRecipientCardProps) {
  const manualRecipient = manualGift?.recipient;
  const isManualDistribution = manualGift?.manual_distribution ?? false;
  const isAssigned = isManualDistribution
    ? Boolean(manualRecipient)
    : Boolean(automaticRecipient);

  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5 dark:border-white/[0.05] dark:bg-white/[0.03]">
      <div className="mb-3 flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold text-gray-800 dark:text-white/90">
          Получатель
        </h2>
        <Badge color={isAssigned ? 'success' : 'warning'} size="sm">
          {isAssigned ? 'Назначен' : 'Не назначен'}
        </Badge>
      </div>

      {isManualDistribution ? (
        manualRecipient ? (
          <div className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
            <p className="font-medium text-gray-800 dark:text-white/90">
              {manualRecipient.display_name}
            </p>
            {manualRecipient.username && <p>@{manualRecipient.username}</p>}
            <p>ID участника: {manualRecipient.id}</p>
          </div>
        ) : (
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Получатель ещё не выбран.
          </p>
        )
      ) : automaticRecipient ? (
        <div className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
          <p className="font-medium text-gray-800 dark:text-white/90">
            {automaticRecipient.participantName}
          </p>
          <p>ID участника: {automaticRecipient.participantID}</p>
        </div>
      ) : (
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Получатель ещё не определён по результатам заезда.
        </p>
      )}
    </div>
  );
}
