<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useGameStore } from '../stores/gameStore';
import { useWS } from '../composables/useWS';
import DashboardManager from '../game/DashboardManager';
import type { Player } from '../types';

const route = useRoute();
const router = useRouter();
const gameStore = useGameStore();
const ws = useWS();
const dashboardManager = new DashboardManager();
const gamePhase = ref('waiting');
const isStarting = ref(false);
const errorMessage = ref<string | null>(null);

// Global game timer (5 minutes = 300 seconds)
const GAME_DURATION = 300;
const gameTimer = ref(GAME_DURATION);
let timerInterval: ReturnType<typeof setInterval> | null = null;

// Check if we have enough players to start
const canStartGame = computed(() => {
  const playerCount = Object.keys(gameStore.players).length;
  return playerCount >= 1 && !isStarting.value;
});

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

// Format timer as MM:SS
function formatTime(seconds: number): string {
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
}

// Start the game timer
function startGameTimer() {
  if (timerInterval) {
    clearInterval(timerInterval);
  }
  
  gameTimer.value = GAME_DURATION;
  // Send initial timer to server
  ws.send('updateTimer', { timer: gameTimer.value });
  
  timerInterval = setInterval(() => {
    if (gameTimer.value > 0) {
      gameTimer.value--;
      // Send timer update to server so players can see it
      ws.send('updateTimer', { timer: gameTimer.value });
    } else {
      // Timer reached 0, end the game
      endGame();
    }
  }, 1000);
}

// Stop the game timer
function stopGameTimer() {
  if (timerInterval) {
    clearInterval(timerInterval);
    timerInterval = null;
  }
}

// End the game
function endGame() {
  stopGameTimer();
  ws.send('endGame', {});
  gamePhase.value = 'ended';
}

// Initialize dashboard
onMounted(async () => {
  const code = route.params.code as string;

  // Validate room code
  if (!code) {
    console.error('No room code provided');
    return;
  }

  // Initialize WebSocket connection first
  ws.init();

  // Setup WebSocket listeners before sending rejoin
  ws.on('rejoinedDashboard', (data: any) => {
    console.log('Dashboard rejoined successfully:', data);
    if (data.gameState) {
      gameStore.setGameState(data.gameState);
      dashboardManager.handleGameUpdate(data.gameState);
      if (data.gameState.gamePhase) {
        gamePhase.value = data.gameState.gamePhase;
        // If rejoining during an active game, start the timer
        if (data.gameState.gamePhase === 'playing' && !timerInterval) {
          startGameTimer();
        }
      }
    }
    // Store terrain data for dashboard to use
    if (data.mapData) {
      sessionStorage.setItem('mapData', JSON.stringify(data.mapData));
    }
  });

  ws.on('gameUpdate', (data: any) => {
    console.log('Dashboard received gameUpdate:', data);
    if (data.gameState) {
      gameStore.setGameState(data.gameState);
      dashboardManager.handleGameUpdate(data.gameState);
      if (data.gameState.gamePhase) {
        gamePhase.value = data.gameState.gamePhase;
      }
    } else if (data) {
      // Handle case where gameState is at root level
      gameStore.setGameState(data);
      dashboardManager.handleGameUpdate(data);
      if (data.gamePhase) {
        gamePhase.value = data.gamePhase;
      }
    }
  });

  ws.on('initialState', (data: any) => {
    console.log('Dashboard received initialState:', data);
    if (data.gameState) {
      gameStore.setGameState(data.gameState);
      dashboardManager.handleGameUpdate(data.gameState);
      if (data.gameState.gamePhase) {
        gamePhase.value = data.gameState.gamePhase;
      }
    } else if (data) {
      gameStore.setGameState(data);
      dashboardManager.handleGameUpdate(data);
      if (data.gamePhase) {
        gamePhase.value = data.gamePhase;
      }
    }
  });

  ws.on('gameStarted', (data: any) => {
    console.log('Game started:', data);
    gamePhase.value = 'playing';
    // Start the 5-minute game timer
    startGameTimer();
  });

  ws.on('gameEnded', (data: any) => {
    console.log('Game ended:', data);
    gamePhase.value = 'ended';
    stopGameTimer();
  });

  // Listen for bullet events and forward to dashboard
  ws.on('bulletSpawn', (data: any) => {
    console.log('Dashboard received bulletSpawn:', data);
    dashboardManager.handleBulletSpawn(data);
  });

  ws.on('bulletDestroy', (data: any) => {
    console.log('Dashboard received bulletDestroy:', data);
    dashboardManager.handleBulletDestroy(data);
  });

  ws.on('playerHit', (data: any) => {
    console.log('Dashboard received playerHit:', data);
    dashboardManager.handlePlayerHit(data);
  });

  ws.on('playerDeath', (data: any) => {
    console.log('Dashboard received playerDeath:', data);
    dashboardManager.handlePlayerDeath(data);
  });

  ws.on('playerRespawn', (data: any) => {
    console.log('Dashboard received playerRespawn:', data);
    dashboardManager.handlePlayerRespawn(data);
  });

  // Handle errors from the server
  ws.on('error', (data: any) => {
    console.error('Dashboard received error:', data);
    if (data.error === 'Room not found') {
      errorMessage.value = 'Room not found. The room may have been closed or the code is invalid.';
      // Redirect to home after showing error
      setTimeout(() => {
        router.push('/');
      }, 3000);
    } else {
      errorMessage.value = data.error || 'An unexpected error occurred';
    }
  });

  // Send rejoin request for dashboard
  setTimeout(() => {
    console.log('Sending dashboard rejoin request:', { code });
    ws.send('rejoinDashboard', { code });
  }, 200);

  // Initialize the Phaser game for spectator view
  dashboardManager.init(ws);
});

onUnmounted(() => {
  stopGameTimer();
  dashboardManager.destroy();
  ws.close();
});

// Focus on a specific player when clicking leaderboard
function focusOnPlayer(playerId: string) {
  dashboardManager.focusOnPlayer(playerId);
}

// Start the game with countdown
function startGame() {
  if (isStarting.value) return; // Prevent multiple clicks

  isStarting.value = true;

  // Play countdown on dashboard, then start the actual game
  dashboardManager.playCountdownAndStart(() => {
    ws.send('startGame', {});
    isStarting.value = false;
    // Start the 5-minute game timer immediately
    gamePhase.value = 'playing';
    startGameTimer();
  });
}
</script>

<template>
  <div class="modern-dashboard w-screen h-screen bg-slate-950 flex flex-col overflow-hidden">
    <!-- Error Modal -->
    <div v-if="errorMessage" class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm">
      <div class="bg-slate-900 border border-red-500/50 rounded-2xl p-8 max-w-md mx-4 shadow-2xl shadow-red-500/20">
        <div class="flex items-center gap-4 mb-4">
          <div class="w-12 h-12 rounded-full bg-red-500/20 flex items-center justify-center">
            <svg class="w-6 h-6 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <h2 class="text-xl font-bold text-white">Error</h2>
        </div>
        <p class="text-slate-300 mb-6">{{ errorMessage }}</p>
        <div class="flex items-center justify-between">
          <span class="text-slate-500 text-sm">Redirecting to home...</span>
          <button 
            @click="router.push('/')" 
            class="px-4 py-2 bg-red-500 hover:bg-red-400 text-white font-semibold rounded-lg transition-colors"
          >
            Go Home Now
          </button>
        </div>
      </div>
    </div>

    <!-- Ambient background glow -->
    <div class="fixed inset-0 pointer-events-none overflow-hidden">
      <div class="absolute -top-40 -left-40 w-96 h-96 bg-purple-600/20 rounded-full blur-3xl"></div>
      <div class="absolute -bottom-40 -right-40 w-96 h-96 bg-cyan-600/20 rounded-full blur-3xl"></div>
    </div>

    <!-- Header Bar -->
    <header class="h-16 bg-slate-900/80 backdrop-blur-xl border-b border-slate-700/50 flex items-center justify-between px-6 shrink-0 relative z-10">
      <!-- Logo & Title -->
      <div class="flex items-center gap-4">
        <div class="w-11 h-11 rounded-xl bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center shadow-lg shadow-violet-500/25">
          <span class="text-white text-2xl font-bold">W</span>
        </div>
        <div class="flex flex-col">
          <h1 class="text-3xl font-bold tracking-wide uppercase">
            <span class="bg-gradient-to-r from-cyan-400 to-blue-500 bg-clip-text text-transparent">WAY</span>
            <span class="bg-gradient-to-r from-fuchsia-400 to-pink-500 bg-clip-text text-transparent">WAR</span>
          </h1>
          <div class="flex items-center gap-2">
            <span class="live-indicator flex items-center gap-1.5 text-sm text-emerald-400 font-semibold uppercase">
              <span class="w-2 h-2 bg-emerald-400 rounded-full animate-pulse"></span>
              Live
            </span>
            <span class="h-4 w-0.5 bg-slate-500 rounded-full"></span>
            <span class="text-slate-400 text-sm font-medium">Room: <span class="text-cyan-400 font-semibold">{{ route.params.code }}</span></span>
          </div>
        </div>
      </div>

      <!-- Center: Start Button or Game Status -->
      <div class="flex items-center gap-4">
        <!-- Start Game Button (shows when game is waiting) -->
        <button
          v-if="gamePhase === 'waiting'"
          @click="startGame"
          :disabled="!canStartGame"
          :class="{
            'bg-gradient-to-r from-emerald-500 to-cyan-500 hover:from-emerald-400 hover:to-cyan-400 shadow-lg shadow-emerald-500/30': canStartGame,
            'bg-gradient-to-r from-slate-600 to-slate-700 cursor-not-allowed opacity-60': !canStartGame
          }"
          class="px-8 py-3 rounded-xl text-white font-bold text-lg tracking-wide transition-all flex items-center gap-3"
        >
          <svg v-if="!isStarting" class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <svg v-else class="w-6 h-6 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" stroke-dasharray="31.4 31.4" />
          </svg>
          {{ isStarting ? 'STARTING...' : 'START GAME' }}
        </button>

        <!-- Game Status (shows when game is playing) -->
        <div v-if="gamePhase === 'playing'" class="bg-emerald-500/10 backdrop-blur-sm px-5 py-2.5 rounded-xl border border-emerald-500/30 flex items-center gap-3">
          <div class="w-3 h-3 bg-emerald-400 rounded-full animate-pulse"></div>
          <span class="text-emerald-400 font-bold text-lg uppercase tracking-wide">Game In Progress</span>
        </div>

        <!-- Game Ended Status -->
        <div v-if="gamePhase === 'ended'" class="bg-red-500/10 backdrop-blur-sm px-5 py-2.5 rounded-xl border border-red-500/30 flex items-center gap-3">
          <div class="w-3 h-3 bg-red-400 rounded-full"></div>
          <span class="text-red-400 font-bold text-lg uppercase tracking-wide">Game Ended</span>
        </div>
      </div>

      <!-- Global Timer -->
      <div 
        class="flex items-center gap-3 backdrop-blur-sm px-5 py-2.5 rounded-xl border"
        :class="{
          'bg-slate-800/60 border-slate-700/50': gameTimer > 60,
          'bg-red-900/60 border-red-500/50': gameTimer <= 60 && gameTimer > 0,
          'bg-red-900/80 border-red-500/70': gameTimer === 0
        }"
      >
        <svg class="w-5 h-5" :class="gameTimer <= 60 ? 'text-red-400' : 'text-amber-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <div 
          class="text-3xl font-bold tabular-nums tracking-wider"
          :class="{
            'text-white': gameTimer > 60,
            'text-red-400 animate-pulse': gameTimer <= 60
          }"
        >
          {{ formatTime(gameTimer) }}
        </div>
      </div>
    </header>

    <!-- Main Content Area -->
    <div class="flex-1 relative">
      <!-- Phaser Game Container -->
      <div id="dashboard-container" class="absolute inset-0 bg-slate-950"></div>

      <!-- Floating Leaderboard - Modern Glass Style -->
      <div class="absolute top-4 right-4 w-80">
        <div class="bg-slate-900/70 backdrop-blur-xl rounded-2xl border border-slate-700/50 overflow-hidden shadow-2xl shadow-black/50">
          <!-- Leaderboard Header -->
          <div class="bg-gradient-to-r from-violet-600 to-fuchsia-600 px-5 py-3">
            <div class="flex items-center gap-2">
              <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
              <span class="text-white font-bold text-lg tracking-wide uppercase">Leaderboard</span>
            </div>
          </div>

          <!-- Leaderboard List -->
          <div class="max-h-80 overflow-y-auto">
            <div
              v-for="player in leaderboard"
              :key="player.id"
              class="flex items-center gap-3 px-4 py-3 cursor-pointer transition-all duration-200 border-b border-slate-800/50 hover:bg-slate-800/50"
              @click="focusOnPlayer(player.id)"
            >
              <!-- Rank Badge -->
              <div
                class="w-8 h-8 rounded-lg flex items-center justify-center text-base font-bold shrink-0"
                :class="{
                  'bg-gradient-to-br from-yellow-400 to-amber-500 text-black shadow-lg shadow-amber-500/30': player.rank === 1,
                  'bg-gradient-to-br from-slate-300 to-slate-400 text-black': player.rank === 2,
                  'bg-gradient-to-br from-orange-400 to-orange-500 text-black': player.rank === 3,
                  'bg-slate-800 text-slate-400': player.rank > 3
                }"
              >
                {{ player.rank }}
              </div>

              <!-- Player Color Indicator -->
              <div
                class="w-4 h-4 rounded-full shrink-0 ring-2 ring-white/20"
                :style="{ backgroundColor: player.color }"
              ></div>

              <!-- Player Name -->
              <div class="flex-1 min-w-0">
                <div class="text-white font-semibold text-base truncate">{{ player.name }}</div>
              </div>

              <!-- Score -->
              <div class="text-right shrink-0">
                <span class="text-emerald-400 font-bold text-lg tabular-nums">{{ player.score.toLocaleString() }}</span>
              </div>
            </div>

            <!-- Empty state -->
            <div v-if="leaderboard.length === 0" class="px-4 py-8 text-center">
              <div class="w-14 h-14 mx-auto mb-3 rounded-full bg-slate-800 flex items-center justify-center">
                <svg class="w-7 h-7 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                </svg>
              </div>
              <div class="text-slate-400 font-semibold text-base">Waiting for players...</div>
              <div class="text-slate-600 text-sm mt-1">Players will appear here when they join</div>
            </div>
          </div>

          <!-- Footer -->
          <div class="bg-slate-800/50 px-4 py-2.5 border-t border-slate-700/50">
            <div class="flex justify-between items-center">
              <span class="text-slate-500 text-sm font-semibold">{{ leaderboard.length }} Players</span>
              <div class="flex items-center gap-1.5">
                <span class="w-2 h-2 bg-emerald-500 rounded-full animate-pulse"></span>
                <span class="text-emerald-400 text-sm font-semibold">Live</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Spectator Badge -->
      <div class="absolute bottom-4 left-4 bg-slate-900/70 backdrop-blur-xl px-5 py-2.5 rounded-full border border-slate-700/50 flex items-center gap-2">
        <svg class="w-5 h-5 text-fuchsia-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
        </svg>
        <span class="text-fuchsia-400 text-sm font-bold tracking-wide uppercase">Spectator Mode</span>
      </div>

      <!-- Game Ended Overlay -->
      <div v-if="gamePhase === 'ended'" class="absolute inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50">
        <div class="bg-slate-900/90 backdrop-blur-xl rounded-3xl border border-slate-700/50 p-8 max-w-lg w-full mx-4 shadow-2xl">
          <div class="text-center mb-6">
            <div class="w-20 h-20 mx-auto mb-4 rounded-full bg-gradient-to-br from-amber-400 to-orange-500 flex items-center justify-center shadow-lg shadow-amber-500/30">
              <svg class="w-10 h-10 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
              </svg>
            </div>
            <h2 class="text-4xl font-bold text-white mb-2 uppercase tracking-wider">Game Over!</h2>
            <p class="text-slate-400 text-lg">Final Standings</p>
          </div>

          <!-- Top 3 Players -->
          <div class="space-y-3 mb-6">
            <div
              v-for="(player, index) in leaderboard.slice(0, 3)"
              :key="player.id"
              class="flex items-center gap-4 p-4 rounded-xl"
              :class="{
                'bg-gradient-to-r from-amber-500/20 to-orange-500/20 border border-amber-500/30': index === 0,
                'bg-gradient-to-r from-slate-400/20 to-slate-500/20 border border-slate-400/30': index === 1,
                'bg-gradient-to-r from-orange-600/20 to-orange-700/20 border border-orange-600/30': index === 2
              }"
            >
              <!-- Rank -->
              <div
                class="w-10 h-10 rounded-full flex items-center justify-center text-lg font-bold shrink-0"
                :class="{
                  'bg-gradient-to-br from-amber-400 to-orange-500 text-black': index === 0,
                  'bg-gradient-to-br from-slate-300 to-slate-400 text-black': index === 1,
                  'bg-gradient-to-br from-orange-500 to-orange-600 text-black': index === 2
                }"
              >
                {{ index + 1 }}
              </div>

              <!-- Player Info -->
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <div
                    class="w-4 h-4 rounded-full shrink-0"
                    :style="{ backgroundColor: player.color }"
                  ></div>
                  <span class="text-white font-bold text-lg truncate">{{ player.name }}</span>
                </div>
              </div>

              <!-- Score -->
              <div class="text-right">
                <span class="text-2xl font-bold" :class="{
                  'text-amber-400': index === 0,
                  'text-slate-300': index === 1,
                  'text-orange-400': index === 2
                }">{{ player.score.toLocaleString() }}</span>
                <span class="text-slate-500 text-sm ml-1">pts</span>
              </div>
            </div>
          </div>

          <!-- Return Home Button -->
          <button
            @click="router.push('/')"
            class="w-full py-4 bg-gradient-to-r from-violet-600 to-fuchsia-600 hover:from-violet-500 hover:to-fuchsia-500 text-white font-bold text-lg rounded-xl transition-all shadow-lg shadow-violet-500/30"
          >
            Return to Home
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Rajdhani:wght@400;500;600;700&display=swap');

.modern-dashboard {
  font-family: 'Rajdhani', sans-serif;
}

/* Live indicator pulse */
.live-indicator span {
  animation: live-pulse 2s ease-in-out infinite;
}

@keyframes live-pulse {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.5;
    transform: scale(0.9);
  }
}

/* Custom scrollbar for leaderboard */
.max-h-80::-webkit-scrollbar {
  width: 4px;
}

.max-h-80::-webkit-scrollbar-track {
  background: transparent;
}

.max-h-80::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.3);
  border-radius: 2px;
}

.max-h-80::-webkit-scrollbar-thumb:hover {
  background: rgba(148, 163, 184, 0.5);
}

/* Smooth hover transitions */
.transition-all {
  transition-property: all;
  transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
}
</style>