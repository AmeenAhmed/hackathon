<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';

// Global timer state
const globalTimer = ref(300); // 5 minutes in seconds
let timerInterval: ReturnType<typeof setInterval> | null = null;

// Leaderboard data (placeholder)
const leaderboard = ref([
  { rank: 1, name: 'PlayerOne', score: 2450, color: '#FFD700' },
  { rank: 2, name: 'ShadowNinja', score: 2180, color: '#C0C0C0' },
  { rank: 3, name: 'DragonSlayer', score: 1950, color: '#CD7F32' },
  { rank: 4, name: 'CosmicKnight', score: 1720, color: '' },
  { rank: 5, name: 'StormBringer', score: 1580, color: '' },
  { rank: 6, name: 'PhantomX', score: 1340, color: '' },
  { rank: 7, name: 'NightHawk', score: 1120, color: '' },
  { rank: 8, name: 'CyberWolf', score: 980, color: '' },
]);

// Format timer as MM:SS
function formatTime(seconds: number): string {
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
}

// Start the countdown timer
onMounted(() => {
  timerInterval = setInterval(() => {
    if (globalTimer.value > 0) {
      globalTimer.value--;
    }
  }, 1000);
});

onUnmounted(() => {
  if (timerInterval) {
    clearInterval(timerInterval);
  }
});
</script>

<template>
  <div class="w-screen h-screen bg-slate-900 flex flex-col overflow-hidden">
    <!-- Header Bar -->
    <header class="h-16 bg-gradient-to-r from-slate-800 via-slate-800 to-slate-800 border-b border-slate-700 flex items-center justify-between px-6 shrink-0">
      <!-- Logo & Title -->
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-lg flex items-center justify-center shadow-lg shadow-indigo-500/30">
          <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
        </div>
        <h1 class="text-xl font-bold text-white tracking-wide">
          <span class="text-indigo-400">Way</span><span class="text-purple-400">War</span>
        </h1>
      </div>

      <!-- Global Timer -->
      <div class="flex items-center gap-3 bg-slate-700/50 px-4 py-2 rounded-lg border border-slate-600">
        <div class="flex items-center gap-2">
          <svg class="w-5 h-5 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span class="text-slate-400 text-sm font-medium">Round Timer</span>
        </div>
        <div class="text-2xl font-mono font-bold text-white tabular-nums">
          {{ formatTime(globalTimer) }}
        </div>
      </div>

      <!-- Player Info (placeholder) -->
      <div class="flex items-center gap-3">
        <div class="w-8 h-8 bg-gradient-to-br from-emerald-400 to-cyan-500 rounded-full flex items-center justify-center text-white font-bold text-sm">
          P
        </div>
        <span class="text-slate-300 font-medium">Player</span>
      </div>
    </header>

    <!-- Main Content Area -->
    <div class="flex-1 relative">
      <!-- Game Scene Placeholder -->
      <div class="absolute inset-0 flex items-center justify-center bg-slate-900">
        <div class="text-center">
          <div class="w-32 h-32 mx-auto mb-6 bg-slate-800 rounded-2xl border-2 border-dashed border-slate-600 flex items-center justify-center">
            <svg class="w-16 h-16 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <h2 class="text-2xl font-bold text-slate-400 mb-2">Game Scene</h2>
          <p class="text-slate-500">The battle arena will render here</p>
        </div>
      </div>

      <!-- Floating Leaderboard -->
      <div class="absolute top-4 right-4 w-72 bg-slate-800/95 backdrop-blur-sm rounded-xl border border-slate-700 shadow-2xl shadow-black/50 overflow-hidden">
        <!-- Leaderboard Header -->
        <div class="bg-gradient-to-r from-indigo-600 to-purple-600 px-4 py-3 flex items-center gap-2">
          <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
          <span class="text-white font-semibold">Leaderboard</span>
        </div>

        <!-- Leaderboard List -->
        <div class="divide-y divide-slate-700/50">
          <div
            v-for="player in leaderboard"
            :key="player.rank"
            class="flex items-center gap-3 px-4 py-2.5 hover:bg-slate-700/30 transition-colors"
          >
            <!-- Rank -->
            <div
              class="w-7 h-7 rounded-full flex items-center justify-center text-sm font-bold shrink-0"
              :class="{
                'bg-amber-500/20 text-amber-400': player.rank === 1,
                'bg-slate-400/20 text-slate-300': player.rank === 2,
                'bg-orange-600/20 text-orange-400': player.rank === 3,
                'bg-slate-700 text-slate-400': player.rank > 3
              }"
            >
              {{ player.rank }}
            </div>

            <!-- Player Name -->
            <div class="flex-1 min-w-0">
              <div class="text-white font-medium text-sm truncate">{{ player.name }}</div>
            </div>

            <!-- Score -->
            <div class="text-right shrink-0">
              <span class="text-emerald-400 font-semibold text-sm tabular-nums">{{ player.score.toLocaleString() }}</span>
            </div>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>