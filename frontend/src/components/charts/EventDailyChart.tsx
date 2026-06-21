"use client";
import React from "react";

import { ApexOptions } from "apexcharts";

import dynamic from "next/dynamic";
// Dynamically import the ReactApexChart component
const ReactApexChart = dynamic(() => import("react-apexcharts"), {
  ssr: false,
});

interface EventDailyChartProps {
  title: string;
  categories: string[]; // подписи по оси X (короткие даты)
  data: number[]; // количество за каждый день
  color?: string;
}

export default function EventDailyChart({
  title,
  categories,
  data,
  color = "#465fff",
}: EventDailyChartProps) {
  const isEmpty = data.length === 0;

  const options: ApexOptions = {
    colors: [color],
    chart: {
      fontFamily: "Outfit, sans-serif",
      type: "bar",
      height: 220,
      toolbar: {
        show: false,
      },
    },
    plotOptions: {
      bar: {
        horizontal: false,
        columnWidth: "45%",
        borderRadius: 4,
        borderRadiusApplication: "end",
      },
    },
    dataLabels: {
      enabled: false,
    },
    stroke: {
      show: true,
      width: 2,
      colors: ["transparent"],
    },
    xaxis: {
      categories,
      axisBorder: {
        show: false,
      },
      axisTicks: {
        show: false,
      },
      labels: {
        rotate: -45,
        hideOverlappingLabels: true,
      },
    },
    legend: {
      show: false,
    },
    yaxis: {
      title: {
        text: undefined,
      },
      labels: {
        formatter: (val: number) => `${Math.round(val)}`,
      },
    },
    grid: {
      yaxis: {
        lines: {
          show: true,
        },
      },
    },
    fill: {
      opacity: 1,
    },
    tooltip: {
      x: {
        show: true,
      },
      y: {
        formatter: (val: number) => `${val}`,
      },
    },
  };

  const series = [
    {
      name: title,
      data,
    },
  ];

  // Ширина подстраивается под число дней: короткие события не растягиваем,
  // длинные — прокручиваем по горизонтали (как BarChartOne).
  const minWidth = Math.max(categories.length * 48, 320);

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
      <h3 className="mb-3 text-lg font-semibold text-gray-800 dark:text-white">{title}</h3>
      {isEmpty ? (
        <div className="flex h-[220px] items-center justify-center text-sm text-gray-500 dark:text-gray-400">
          Нет данных
        </div>
      ) : (
        <div className="max-w-full overflow-x-auto custom-scrollbar">
          <div style={{ minWidth }}>
            <ReactApexChart
              options={options}
              series={series}
              type="bar"
              height={220}
            />
          </div>
        </div>
      )}
    </div>
  );
}
