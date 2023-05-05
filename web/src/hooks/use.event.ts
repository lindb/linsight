import { PlatformStore } from '@src/stores';
import { reaction } from 'mobx';
import { useEffect, useState } from 'react';
import { MouseEvent } from '@src/types';

export function useMouseEvent() {
  const [mouseEvent, setMouseEvent] = useState<MouseEvent | null>(null);

  useEffect(() => {
    const disposer = reaction(
      () => PlatformStore.mouseEvent,
      () => {
        setMouseEvent(PlatformStore.mouseEvent);
      }
    );

    return () => disposer();
  }, []);

  return { mouseEvent };
}
