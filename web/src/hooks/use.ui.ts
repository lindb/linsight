/*
Licensed to LinDB under one or more contributor
license agreements. See the NOTICE file distributed with
this work for additional information regarding copyright
ownership. LinDB licenses this file to you under
the Apache License, Version 2.0 (the "License"); you may
not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
 
Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/
import { MutableRefObject, useEffect, useMemo, useState } from 'react';

/*
 * Lazy load hook
 */
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
