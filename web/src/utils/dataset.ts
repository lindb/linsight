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
import { isEmpty, keys, forIn, get, trim, find } from 'lodash-es';
import { ColorKit } from '@src/utils';
import moment from 'moment';
import { DataSetType, PanelSetting, Query } from '@src/types';

const getGroupByTags = (tags?: any): string => {
  if (!tags) {
    return '';
  }
  const tagKeys = Object.keys(tags);
  if (tagKeys.length === 1) {
    return tags[tagKeys[0]];
  }
  const result = [];
  for (let key of tagKeys) {
    result.push(`${key}:${tags[key]}`);
  }
  return result.join(',');
};

const formatLegend = (target?: Query, tags?: object): string => {
  const format = get(target, 'legendFormat', '');
  if (!isEmpty(format)) {
    return format.replace(/{{\s*(\w+)\s*}}/g, (_, key) => get(tags, trim(key), ''));
  }
  return getGroupByTags(tags);
};

const createStatsDatasets = (resultSet: any, panel: PanelSetting): any => {
  const datasets: any[] = [];
  let colorIdx = 0;
  const targets = get(panel, 'targets', []);
  forIn(resultSet, (rs: any, refId: string) => {
    if (!rs) {
      return;
    }
    const { series } = rs;

    if (isEmpty(series)) {
      return;
    }
    const target = find(targets, { refId: refId });
    series.forEach((item: any) => {
      const { tags, fields } = item;
      if (!fields) {
        return;
      }
      const tagLabel = formatLegend(target, tags);
      forIn(fields, (field: any, key: string) => {
        const bgColor = ColorKit.getColor(colorIdx++);
        const label = target?.includeField ? `${isEmpty(tagLabel) ? key : `${tagLabel}:${key}`}` : tagLabel;
        const points: { [timestamp: string]: number } = field;
        const timestamps = keys(points);
        let val = 0;
        if (!isEmpty(timestamps)) {
          const value = points[`${timestamps[0]}`];
          val = value ? Math.floor(value * 1000) / 1000 : 0;
        }
        datasets.push({ label: label, value: val, backgroundColor: bgColor });
      });
    });
  });
  return datasets;
};

const createTimeSeriesDatasets = (resultSet: any, panel: PanelSetting): any => {
  const datasets: any[] = [];
  //TODO: calc min interval/max time range
  let timeCtx = {
    startTime: 0,
    endTime: 0,
    interval: 0,
  };
  let colorIdx = 0;
  const targets = get(panel, 'targets', []);
  forIn(resultSet, (rs: any, refId: string) => {
    if (!rs) {
      return;
    }
    const { series, startTime, endTime, interval, metricName } = rs;

    if (isEmpty(series)) {
      return;
    }
    const target = find(targets, { refId: refId });
    timeCtx = { startTime, endTime, interval };
    series.forEach((item: any) => {
      const { tags, fields } = item;
      if (!fields) {
        return;
      }
      const tagLabel = formatLegend(target, tags);
      forIn(fields, (field: any, key: string) => {
        const bgColor = ColorKit.getColor(colorIdx++);

        const borderColor = bgColor;
        const pointBackgroundColor = ColorKit.toRGBA(bgColor, 0.25);
        const label = target?.includeField ? `${isEmpty(tagLabel) ? key : `${tagLabel}:${key}`}` : tagLabel;

        let data: any = [];
        const points: { [timestamp: string]: number } = field;
        let i = 0;
        let timestamp = startTime! + i * interval!;
        const stats = {
          count: 0,
          total: 0,
          mean: 0,
          max: -Infinity,
          min: Infinity,
          last: Infinity,
          first: Infinity,
        };
        for (; timestamp <= endTime!; ) {
          const value = points[`${timestamp}`];
          // FIXME: need handl null
          const v = value ? Math.floor(value * 1000) / 1000 : 0;
          if (value !== null) {
            stats.count++;
            stats.total += v;
            stats.max = Math.max(stats.max, v);
            stats.min = Math.min(stats.min, v);
            stats.last = v;
            if (stats.first === Infinity) {
              // set first value if not exist
              stats.first = v;
            }
          }

          data.push(v);
          i++;
          timestamp = startTime! + i * interval!;
        }
        stats.mean = Math.floor((stats.total * 1000) / stats.count) / 1000;

        datasets.push({
          label,
          refId,
          data,
          borderColor,
          pointBackgroundColor,
          hidden: false,
          stats,
          meta: {
            metricName,
            tags,
            field: key,
          },
        });
      });
    });
  });
  if (isEmpty(datasets)) {
    // no data in response
    return;
  }
  const labels = [];
  const { startTime, endTime, interval } = timeCtx;
  const start = new Date(startTime!);
  const end = new Date(endTime!);
  const showTimeLabel =
    start.getDate() !== end.getDate() ||
    start.getMonth() !== end.getMonth() ||
    start.getFullYear() !== end.getFullYear();
  const range = endTime! - startTime!;
  let i = 0;
  let timestamp = startTime! + i * interval!;
  const times = [];
  const timeLabels = [];
  for (; timestamp <= endTime!; ) {
    const dateTime = moment(timestamp);
    if (showTimeLabel) {
      labels.push(dateTime.format('MM/DD HH:mm'));
    } else if (range > 5 * 60 * 1000) {
      labels.push(dateTime.format('HH:mm'));
    } else {
      labels.push(dateTime.format('HH:mm:ss'));
    }
    timeLabels.push(dateTime.format('YYYY-MM-DD HH:mm:ss'));
    times.push(timestamp);
    i++;
    timestamp = startTime! + i * interval!;
  }
  return { labels, datasets, interval, times, timeLabels };
};

const createDatasets = (resultSet: any, type: DataSetType, panel: PanelSetting): any => {
  switch (type) {
    case DataSetType.SingleStat:
      return createStatsDatasets(resultSet, panel);
    default:
      return createTimeSeriesDatasets(resultSet, panel);
  }
};

export default {
  createDatasets,
};
