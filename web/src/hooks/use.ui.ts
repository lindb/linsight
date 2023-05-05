import { MutableRefObject, useEffect, useMemo, useState } from 'react';

export function useIsInViewport(ref: MutableRefObject<HTMLElement>) {
  const [isIntersecting, setIsIntersecting] = useState(false);

  const observer = useMemo(() => {
    const io = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting) {
        setIsIntersecting(true);
        // if isIntersecting disable observe
        io.unobserve(entry.target);
      }
    });
    return io;
  }, []);

  useEffect(() => {
    const target = ref.current;
    if (!target) {
      return;
    }
    observer.observe(target);

    return () => {
      observer.disconnect();
    };
  }, [ref, observer]);

  return { isIntersecting };
}
