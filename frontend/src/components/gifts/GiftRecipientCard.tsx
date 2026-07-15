import Badge from '@/components/ui/badge/Badge';
import type { ManualGift } from '@/types';

interface GiftRecipientCardProps {
  manualGift?: ManualGift;
}

export default function GiftRecipientCard({
  manualGift,
}: GiftRecipientCardProps) {
  const recipient = manualGift?.recipient;

  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5 dark:border-white/[0.05] dark:bg-white/[0.03]">
      <div className="mb-3 flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold text-gray-800 dark:text-white/90">
          Получатель
        </h2>
        {manualGift?.manual_distribution && (
          <Badge color={recipient ? 'success' : 'warning'} size="sm">
            {recipient ? 'Назначен' : 'Не назначен'}
          </Badge>
        )}
      </div>

      {manualGift?.manual_distribution ? (
        recipient ? (
          <div className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
            <p className="font-medium text-gray-800 dark:text-white/90">
              {recipient.display_name}
            </p>
            {recipient.username && <p>@{recipient.username}</p>}
            <p>ID участника: {recipient.id}</p>
          </div>
        ) : (
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Получатель ещё не выбран.
          </p>
        )
      ) : (
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Приз распределяется автоматически по результатам заезда.
        </p>
      )}
    </div>
  );
}
