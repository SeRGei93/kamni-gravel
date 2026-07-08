import { useEffect, useLayoutEffect } from "react";

// useLayoutEffect на клиенте (восстановление скролла до отрисовки — без мигания),
// useEffect на сервере, где useLayoutEffect не работает и React выдаёт warning.
export const useIsomorphicLayoutEffect =
  typeof window !== "undefined" ? useLayoutEffect : useEffect;
