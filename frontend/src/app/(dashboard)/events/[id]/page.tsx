import type { Metadata } from "next";
import React from "react";

export const metadata: Metadata = {
  title: "Детали события | Gravel Bot Admin",
  description: "Детальная информация о событии",
};

export default function EventDetailPage({
  params,
}: {
  params: { id: string };
}) {
  return (
    <div>
      <h1 className="mb-4 text-2xl font-semibold text-gray-800 dark:text-white">
        Событие #{params.id}
      </h1>
      <p className="text-gray-600 dark:text-gray-400">
        Детальная информация о событии будет здесь
      </p>
    </div>
  );
}
