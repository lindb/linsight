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
import { FormatRepositoryInst, PanelSetting } from '@src/types';
import classNames from 'classnames';
import React, { MutableRefObject, useEffect, useRef, useState } from 'react';
import { ColorMode, JustifyMode, StatOptions, TextMode } from '../types';
import { get } from 'lodash-es';

const StatChart: React.FC<{ dataset: any; options: StatOptions; panel: PanelSetting }> = (props) => {
  const { dataset, options, panel } = props;
  const containerRef = useRef() as MutableRefObject<HTMLDivElement>;
  const [fontSize, setFontSize] = useState(14);
  const fieldConfig = get(panel, 'fieldConfig.defaults', {});

  useEffect(() => {
    const resizeObserver = new ResizeObserver((entries) => {
      for (let entry of entries) {
        const { width, height } = entry.contentRect;
        const fontSize = Math.min(width, height) / 8;
        setFontSize(fontSize);
      }
    });

    resizeObserver.observe(containerRef.current);
    return () => {
      resizeObserver.disconnect();
    };
  }, []);
  const renderContent = () => {
    switch (options.textMode) {
      case TextMode.name:
        return <div style={{ fontSize: `${fontSize}px`, lineHeight: 1 }}>{dataset.label}</div>;
      case TextMode.value:
        return (
          <div
            style={{
              color: options.colorMode === ColorMode.value ? dataset.backgroundColor : 'inherit',
              fontSize: `${fontSize * 2}px`,
              lineHeight: 1,
            }}>
            {FormatRepositoryInst.get(get(fieldConfig, 'unit', '')).formatString(dataset.value)}
          </div>
        );
      case TextMode.none:
        return <></>;
      default:
        return (
          <>
            <div style={{ fontSize: `${fontSize}px`, lineHeight: 1 }}>{dataset.label}</div>
            <div
              style={{
                color: options.colorMode === ColorMode.value ? dataset.backgroundColor : 'inherit',
                fontSize: `${fontSize * 2}px`,
                fontWeight: 500,
                lineHeight: 1,
              }}>
              {FormatRepositoryInst.get(get(fieldConfig, 'unit', '')).formatString(dataset.value)}
            </div>
          </>
        );
    }
  };
  const statCls = classNames('stat-container', {
    'text-to-center': options.justifyMode === JustifyMode.center,
    'has-background': options.colorMode === ColorMode.background,
  });
  return (
    <div
      ref={containerRef}
      className={statCls}
      style={{ background: options.colorMode === ColorMode.background ? dataset.backgroundColor : 'none' }}>
      {renderContent()}
    </div>
  );
};

export default StatChart;
