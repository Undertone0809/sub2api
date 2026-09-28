<template>
  <div class="card p-4">
    <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">
      {{ t(useRollingTrend ? 'payment.admin.rollingRevenue' : 'payment.admin.dailyRevenue') }}
    </h3>
    <div v-if="loading" class="flex h-64 items-center justify-center">
      <LoadingSpinner size="md" />
    </div>
    <template v-else-if="useRollingTrend">
      <div v-if="rollingCurrencies.length > 0" class="space-y-5">
        <section
          v-for="currency in rollingCurrencies"
          :key="currency"
          class="min-w-0 border-t border-gray-200 pt-4 first:border-t-0 first:pt-0 dark:border-dark-700"
        >
          <h4 class="mb-2 text-xs font-semibold text-gray-700 dark:text-gray-300">{{ currency }}</h4>
          <div class="h-56">
            <Line :data="rollingChartData(currency)" :options="rollingChartOptions(currency)" />
          </div>
        </section>
      </div>
      <div
        v-else
        class="flex h-64 items-center justify-center text-sm text-gray-500 dark:text-gray-400"
      >
        {{ t('payment.admin.noData') }}
      </div>
    </template>
    <div v-else class="h-64">
      <Line v-if="chartData" :data="chartData" :options="chartOptions" />
      <div v-else class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400">
        {{ t('payment.admin.noData') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { DailyPaymentStats, RevenueTrendPoint } from '@/types/payment'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)

const { t } = useI18n()

const props = defineProps<{
  data: DailyPaymentStats[]
  revenueTrend?: RevenueTrendPoint[]
  loading?: boolean
}>()

const colors = [
  ['rgb(59, 130, 246)', 'rgba(59, 130, 246, 0.1)'],
  ['rgb(168, 85, 247)', 'rgba(168, 85, 247, 0.1)'],
  ['rgb(245, 158, 11)', 'rgba(245, 158, 11, 0.1)'],
  ['rgb(239, 68, 68)', 'rgba(239, 68, 68, 0.1)'],
]

const useRollingTrend = computed(() => props.revenueTrend !== undefined)
const rollingCurrencies = computed(() => {
  const trend = props.revenueTrend || []
  const currencies = [...new Set(trend.flatMap(point => [
    ...Object.keys(point.rolling_7d_amount || {}),
    ...Object.keys(point.rolling_30d_amount || {}),
  ]))].sort()
  return currencies.filter(currency => trend.some(point =>
    typeof point.rolling_7d_amount[currency] === 'number' ||
    typeof point.rolling_30d_amount[currency] === 'number'
  ))
})

const chartData = computed(() => {
  if (!props.data || props.data.length === 0) return null
  const currencies = [...new Set(props.data.flatMap(day => Object.keys(day.amount)))].sort()
  return {
    labels: props.data.map(d => d.date),
    datasets: [
      ...currencies.map((currency, index) => {
        const [borderColor, backgroundColor] = colors[index % colors.length]
        return {
          label: `${currency} ${t('payment.admin.revenue')}`,
          data: props.data.map(day => day.amount[currency] || 0),
          borderColor,
          backgroundColor,
          fill: true,
          tension: 0.3,
          pointRadius: 3,
          pointHoverRadius: 5,
        }
      }),
      {
        label: t('payment.admin.orderCount'),
        data: props.data.map(d => d.count),
        borderColor: 'rgb(16, 185, 129)',
        backgroundColor: 'rgba(16, 185, 129, 0.1)',
        fill: false,
        tension: 0.3,
        pointRadius: 3,
        pointHoverRadius: 5,
        yAxisID: 'y1',
      }
    ]
  }
})

function rollingChartData(currency: string) {
  const trend = props.revenueTrend || []
  const [borderColor] = colors[rollingCurrencies.value.indexOf(currency) % colors.length]
  return {
    labels: trend.map(point => point.date),
    datasets: [
      {
        label: t('payment.admin.rolling7Days'),
        data: trend.map(point => point.rolling_7d_amount[currency] ?? null),
        borderColor,
        backgroundColor: 'transparent',
        borderWidth: 2,
        fill: false,
        tension: 0,
        pointRadius: 0,
        pointHoverRadius: 4,
        spanGaps: false,
      },
      {
        label: t('payment.admin.rolling30Days'),
        data: trend.map(point => point.rolling_30d_amount[currency] ?? null),
        borderColor,
        backgroundColor: 'transparent',
        borderDash: [6, 4],
        borderWidth: 2,
        fill: false,
        tension: 0,
        pointRadius: 0,
        pointHoverRadius: 4,
        spanGaps: false,
      },
    ]
  }
}

function rollingChartOptions(currency: string) {
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: 'index' as const, intersect: false },
    scales: {
      y: {
        type: 'linear' as const,
        display: true,
        beginAtZero: true,
        title: { display: true, text: `${t('payment.admin.revenue')} (${currency})` },
      },
    },
    plugins: {
      legend: { position: 'top' as const },
    },
  }
}

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { mode: 'index' as const, intersect: false },
  scales: {
    y: {
      type: 'linear' as const,
      display: true,
      position: 'left' as const,
      title: { display: true, text: t('payment.admin.revenue') },
    },
    y1: {
      type: 'linear' as const,
      display: true,
      position: 'right' as const,
      title: { display: true, text: t('payment.admin.orderCount') },
      grid: { drawOnChartArea: false },
    }
  },
  plugins: {
    legend: { position: 'top' as const },
  }
}
</script>
