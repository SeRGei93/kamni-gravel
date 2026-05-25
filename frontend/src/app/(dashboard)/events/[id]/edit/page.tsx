import { redirect } from 'next/navigation';

interface EventEditRedirectPageProps {
  params: Promise<{ id: string }>;
}

export default async function EventEditRedirectPage({
  params,
}: EventEditRedirectPageProps) {
  const { id } = await params;
  redirect(`/events/${id}`);
}
