import { redirect } from 'next/navigation';

interface GiftEditRedirectPageProps {
  params: Promise<{ id: string }>;
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
}

function appendSearchParam(
  params: URLSearchParams,
  key: string,
  value: string | string[] | undefined
) {
  if (Array.isArray(value)) {
    value.forEach((item) => params.append(key, item));
    return;
  }

  if (value !== undefined) {
    params.set(key, value);
  }
}

export default async function GiftEditRedirectPage({
  params,
  searchParams,
}: GiftEditRedirectPageProps) {
  const { id } = await params;
  const resolvedSearchParams = searchParams ? await searchParams : {};
  const query = new URLSearchParams();

  Object.entries(resolvedSearchParams).forEach(([key, value]) => {
    appendSearchParam(query, key, value);
  });

  const queryString = query.toString();
  redirect(`/gifts/${id}${queryString ? `?${queryString}` : ''}`);
}
