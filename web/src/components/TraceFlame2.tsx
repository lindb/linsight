import { FlamegraphRenderer, convertJaegerTraceToProfile } from '@pyroscope/flamegraph';
import '@pyroscope/flamegraph/dist/index.css';
import { APMSrv } from '@src/services';
import React from 'react';

const TraceFlame2: React.FC = () => {
  console.log('.....', convertJaegerTraceToProfile(APMSrv.getTraceData2()[0]));
  return (
    <div>
      <FlamegraphRenderer
        profile={convertJaegerTraceToProfile(APMSrv.getTraceData2()[0])}
        onlyDisplay="flamegraph"
        showToolbar={false}
        showCredit={false}
      />
    </div>
  );
};

export default TraceFlame2;
