<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useGameStore } from '../stores/gameStore';
import { useWS } from '../composables/useWS';
import DashboardManager from '../game/DashboardManager';
import type { Player } from '../types';

const gameStore = useGameStore();
const ws = useWS();
const dashboardManager = new DashboardManager();

// Computed leaderboard from game state
const leaderboard = computed(() => {
  const players = Object.values(gameStore.players) as Player[];
  const scores = gameStore.score;
  
  return players
    .map((player, index) => ({
      rank: index + 1,
      id: player.id,
      name: player.name,
      score: scores[player.id] || 0,
      color: player.color
    }))
    .sort((a, b) => b.score - a.score)
    .map((player, index) => ({ ...player, rank: index + 1 }));
});

// Timer from game state
const globalTimer = computed(() => gameStore.timer);

// Format timer as MM:SS
function formatTime(seconds: number): string {
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
}

// Initialize dashboard
onMounted(() => {
  // Initialize WebSocket connection first
  ws.init();

  // Setup WebSocket listener for game updates
  ws.on('gameUpdate', (data: any) => {
    console.log('Dashboard received gameUpdate:', data);
    if (data.gameState) {
      gameStore.setGameState(data.gameState);
      dashboardManager.handleGameUpdate(data.gameState);
    } else if (data) {
      // Handle case where gameState is at root level
      gameStore.setGameState(data);
      dashboardManager.handleGameUpdate(data);
    }
  });

  ws.on('initialState', (data: any) => {
    console.log('Dashboard received initialState:', data);
    if (data.gameState) {
      gameStore.setGameState(data.gameState);
      dashboardManager.handleGameUpdate(data.gameState);
    } else if (data) {
      gameStore.setGameState(data);
      dashboardManager.handleGameUpdate(data);
    }
  });

  // Initialize the Phaser game for spectator view
  dashboardManager.init(ws);
});

onUnmounted(() => {
  dashboardManager.destroy();
  ws.close();
});

// Focus on a specific player when clicking leaderboard
function focusOnPlayer(playerId: string) {
  dashboardManager.focusOnPlayer(playerId);
}
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
      <!-- Phaser Game Container -->
      <div id="dashboard-container" class="absolute inset-0 bg-slate-900"></div>

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
            :key="player.id"
            class="flex items-center gap-3 px-4 py-2.5 hover:bg-slate-700/30 transition-colors cursor-pointer"
            @click="focusOnPlayer(player.id)"
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

            <!-- Player Color Indicator -->
            <div
              class="w-3 h-3 rounded-full shrink-0"
              :style="{ backgroundColor: player.color }"
            ></div>

            <!-- Player Name -->
            <div class="flex-1 min-w-0">
              <div class="text-white font-medium text-sm truncate">{{ player.name }}</div>
            </div>

            <!-- Score -->
            <div class="text-right shrink-0">
              <span class="text-emerald-400 font-semibold text-sm tabular-nums">{{ player.score.toLocaleString() }}</span>
            </div>
          </div>

          <!-- Empty state -->
          <div v-if="leaderboard.length === 0" class="px-4 py-6 text-center text-slate-500 text-sm">
            Waiting for players to join...
          </div>
        </div>

      </div>
    </div>
  </div>
</template>