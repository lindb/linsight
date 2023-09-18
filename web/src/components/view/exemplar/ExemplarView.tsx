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
import { Modal, TagGroup } from '@douyinfe/semi-ui';
import { StatusTip, TraceView } from '@src/components';
import { useRequest } from '@src/hooks';
import { DataQuerySrv } from '@src/services';
import { DatasourceStore, ExemplarStore, TraceViewStore } from '@src/stores';
import { Exemplar, FormatRepositoryInst } from '@src/types';
import { DataSetKit } from '@src/utils';
import { cloneDeep, get, isEmpty, set, unset } from 'lodash-es';
import { reaction } from 'mobx';
import React, { useEffect, useState } from 'react';
import './exemplar-view.scss';

const ExemplarView: React.FC = () => {
  const [visible, setVisible] = useState(false);
  const [exemplars, setExemplars] = useState<Exemplar[]>([]);
  const [selectExemplar, setSelectExemplar] = useState<Exemplar | null>(null);

  const getMetricQueryTarget = () => {
    return get(ExemplarStore.selectDataPoint, 'point.meta.target');
  };

  const { loading, result, error } = useRequest(
    ['load_exemplars', ExemplarStore.selectDataPoint],
    () => {
      const target = cloneDeep(getMetricQueryTarget());
      if (!target) {
        return;
      }
      const metricDatasourceUID = get(target, 'datasource.uid', '');
      const metricDatasource = DatasourceStore.getDatasource(metricDatasourceUID);
      if (!metricDatasource) {
        return null;
      }
      set(target, 'request.fields', ['rpc_e']);
      unset(target, 'request.groupBy');

      const timestamp = get(ExemplarStore.selectDataPoint, 'timestamp', 0);
      if (timestamp <= 0) {
        return;
      }
      const interval = get(ExemplarStore.selectDataPoint, 'point.meta.interval', 0);
      return DataQuerySrv.dataQuery({
        queries: [target],
        range: {
          from: timestamp,
          to: timestamp + interval - 1,
        },
      });
    },
    { enabled: !isEmpty(getMetricQueryTarget()) }
  );

  useEffect(() => {
    const exemplars = DataSetKit.createExemplarDatasets(result);
    setExemplars(exemplars);
    setSelectExemplar(get(exemplars, '[0]', null));
  }, [result]);

  useEffect(() => {
    const disposer = reaction(
      () => ExemplarStore.selectDataPoint,
      (datapoint: any | null) => {
        if (datapoint) {
          setVisible(true);
        } else {
          setVisible(false);
        }
      }
    );

    return () => disposer();
  }, []);

  const renderTrace = () => {
    if (isEmpty(exemplars)) {
      return <div> no data</div>;
    }
    const target = getMetricQueryTarget();
    const metricDatasourceUID = get(target, 'datasource.uid', '');
    const metricDatasource = DatasourceStore.getDatasource(metricDatasourceUID);
    if (!metricDatasource) {
      return <div>no metric datasource</div>;
    }
    const traceDatasources = get(metricDatasource.setting, 'config.traceDatasources', []);
    if (isEmpty(traceDatasources)) {
      return <div>no trace datasource</div>;
    }
    return (
      <TraceView
        newWindowLink
        traceId={exemplars[0].traceId}
        spanId={exemplars[0].spanId}
        datasources={traceDatasources}
      />
    );
  };

  return (
    <Modal
      className="exemplar-modal"
      fullScreen
      closeOnEsc
      visible={visible}
      footer={null}
      title={
        <>
          <TagGroup
            tagList={(exemplars || []).map((item: Exemplar): any => {
              const tagKey = item.traceId + item.spanId;
              const selected = `${selectExemplar?.traceId}${selectExemplar?.spanId}` === tagKey;
              return {
                tagKey: tagKey,
                children: `L(${FormatRepositoryInst.get('ns').formatString(item.duration, 3)})`,
                style: {
                  background: selected ? 'var(--semi-color-primary)' : 'var(--semi-color-tertiary)',
                },
                type: 'solid',
              };
            })}
          />
        </>
      }
      onCancel={() => {
        if (TraceViewStore.viewVisibleSpanView) {
          // if span view is visible, need close span viw
          TraceViewStore.setVisibleSpanView(false);
        } else {
          setVisible(false);
        }
      }}>
      {loading && <StatusTip isLoading={loading} error={error} />}
      {renderTrace()}
    </Modal>
  );
};

export default ExemplarView;
