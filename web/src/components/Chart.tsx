import React, { MutableRefObject, useEffect, useRef } from 'react';
import { Chart, registerables } from 'chart.js';

Chart.register(...registerables);

const CanvasChart: React.FC<{ type: string; data: any; height?: number }> = (props) => {
  const { type, data, height } = props;
  const canvasRef = useRef() as MutableRefObject<HTMLCanvasElement | null>;

  useEffect(() => {
    let chart: any = null;
    if (canvasRef.current) {
      chart = new Chart(canvasRef.current, {
        type: type,
        data: data,
        options: {
          responsive: true,
          maintainAspectRatio: false,
          plugins: {
            legend: {
              display: false,
            },
            title: {
              display: false,
            },
          },
          scales: {
            x: {
              grid: {
                color: 'rgba(232,233,234,1)',
              },
              ticks: {
                color: 'rgba(28,31,35,0.8)',
              },
            },
            y: {
              grid: {
                // color: "rgba(28,31,35,0.2)",
                color: 'rgba(232,233,234,1)',
              },
              ticks: {
                color: 'rgba(28,31,35,0.8)',
              },
            },
          },
          elements: {
            line: {
              tension: 0, // disables bezier curve
              borderWidth: 1,
              fill: undefined,
            },
            point: {
              radius: 0,
              hoverRadius: 0,
              pointStyle: undefined,
            },
            arc: {
              borderWidth: 0,
            },
          },
        },
      } as any);
    }
    return () => {
      if (chart) {
        chart.destroy();
      }
    };
  }, []);

  return (
    <div style={{ height: height || 280, width: '100%' }}>
      <canvas ref={canvasRef} />
    </div>
  );
};

export default CanvasChart;
