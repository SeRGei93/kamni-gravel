import { redirect } from 'next/navigation';

interface GiftEditRedirectPageProps {
  params: Promise<{ id: string }>;
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
}

export default async function GiftEditRedirectPage({
  params,
  searchParams,
}: GiftEditRedirectPageProps) {
  const { id } = await params;
  const resolvedSearchParams = searchParams ? await searchParams : {};
  const query = new URLSearchParams();

  // Сохраняем только поддерживаемое состояние списка призов (статус проверки),
  // чтобы не протаскивать устаревший event_id после отказа от фильтра по событию.
  const reviewStatus = resolvedSearchParams.review_status;
  const reviewStatusValue = Array.isArray(reviewStatus)
    ? reviewStatus[0]
    : reviewStatus;
  if (
    reviewStatusValue === 'pending_review' ||
    reviewStatusValue === 'approved'
  ) {
    query.set('review_status', reviewStatusValue);
  }

  const queryString = query.toString();
  redirect(`/gifts/${id}${queryString ? `?${queryString}` : ''}`);
}
