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
const showInstructions = ref(true);

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

// Computed group accuracy (average of all players' accuracy)
const groupAccuracy = computed(() => {
  const players = Object.values(gameStore.players) as Player[];
  if (players.length === 0) return 0;
  
  let totalCorrect = 0;
  let totalAttempted = 0;
  
  players.forEach(player => {
    totalCorrect += player.correctAnswers || 0;
    totalAttempted += player.questionsAttempted || 0;
  });
  
  if (totalAttempted === 0) return 0;
  return Math.round((totalCorrect / totalAttempted) * 100);
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
  ws.send('updateTimer', { timer: gameTimer.value });
  
  timerInterval = setInterval(() => {
    if (gameTimer.value > 0) {
      gameTimer.value--;
      ws.send('updateTimer', { timer: gameTimer.value });
    } else {
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

  if (!code) {
    console.error('No room code provided');
    return;
  }

  ws.init();

  ws.on('rejoinedDashboard', (data: any) => {
    console.log('Dashboard rejoined successfully:', data);
    if (data.gameState) {
      gameStore.setGameState(data.gameState);
      dashboardManager.handleGameUpdate(data.gameState);
      if (data.gameState.gamePhase) {
        gamePhase.value = data.gameState.gamePhase;
        if (data.gameState.gamePhase === 'playing' && !timerInterval) {
          startGameTimer();
        }
      }
    }
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
    startGameTimer();
  });

  ws.on('gameEnded', (data: any) => {
    console.log('Game ended:', data);
    gamePhase.value = 'ended';
    stopGameTimer();
  });

  ws.on('bulletSpawn', (data: any) => {
    dashboardManager.handleBulletSpawn(data);
  });

  ws.on('bulletDestroy', (data: any) => {
    dashboardManager.handleBulletDestroy(data);
  });

  ws.on('playerHit', (data: any) => {
    dashboardManager.handlePlayerHit(data);
  });

  ws.on('playerDeath', (data: any) => {
    dashboardManager.handlePlayerDeath(data);
  });

  ws.on('playerRespawn', (data: any) => {
    dashboardManager.handlePlayerRespawn(data);
  });

  ws.on('error', (data: any) => {
    console.error('Dashboard received error:', data);
    if (data.error === 'Room not found') {
      errorMessage.value = 'Room not found. The room may have been closed or the code is invalid.';
      setTimeout(() => {
        router.push('/');
      }, 3000);
    } else {
      errorMessage.value = data.error || 'An unexpected error occurred';
    }
  });

  setTimeout(() => {
    console.log('Sending dashboard rejoin request:', { code });
    ws.send('rejoinDashboard', { code });
  }, 200);

  dashboardManager.init(ws);
});

onUnmounted(() => {
  stopGameTimer();
  dashboardManager.destroy();
  ws.close();
});

function focusOnPlayer(playerId: string) {
  dashboardManager.focusOnPlayer(playerId);
}

function copyCode() {
  const code = route.params.code as string;
  navigator.clipboard.writeText(code);
}

function startGame() {
  if (isStarting.value) return;
  isStarting.value = true;

  dashboardManager.playCountdownAndStart(() => {
    ws.send('startGame', {});
    isStarting.value = false;
    gamePhase.value = 'playing';
    startGameTimer();
  });
}
</script>

<template>
  <div class="dashboard w-screen h-screen flex flex-col overflow-hidden">
    <!-- Dark Background -->
    <div class="fixed inset-0 bg-base -z-10"></div>
    
    <!-- Subtle Glow Effects -->
    <div class="fixed inset-0 pointer-events-none overflow-hidden -z-10">
      <div class="absolute top-0 left-1/2 -translate-x-1/2 w-[600px] h-[300px] bg-teal/8 rounded-full blur-3xl"></div>
      <div class="absolute -bottom-40 -left-40 w-[500px] h-[500px] bg-coral/5 rounded-full blur-3xl"></div>
      <div class="absolute -bottom-40 -right-40 w-[500px] h-[500px] bg-gold/5 rounded-full blur-3xl"></div>
    </div>

    <!-- Error Modal -->
    <div v-if="errorMessage" class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm">
      <div class="card-modal rounded-2xl p-8 max-w-md mx-4">
        <div class="flex items-center gap-4 mb-4">
          <div class="w-12 h-12 rounded-full bg-red-100 flex items-center justify-center">
            <svg class="w-6 h-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <h2 class="text-xl font-bold text-dark">Error</h2>
        </div>
        <p class="text-dark/70 mb-6">{{ errorMessage }}</p>
        <div class="flex items-center justify-between">
          <span class="text-dark/40 text-sm">Redirecting to home...</span>
          <button @click="router.push('/')" class="btn-danger px-4 py-2 font-semibold rounded-lg">
            Go Home Now
          </button>
        </div>
      </div>
    </div>

    <!-- Header Bar -->
    <header class="header-bar h-16 flex items-center justify-between px-6 shrink-0 relative z-10">
      <!-- Logo & Title -->
      <div class="flex items-center gap-4">
        <div class="logo-icon w-11 h-11 rounded-xl flex items-center justify-center">
          <span class="text-dark text-2xl font-bold">W</span>
        </div>
        <div class="flex flex-col">
          <h1 class="text-3xl font-bold tracking-wide uppercase">
            <span class="text-white">WAY</span>
            <span class="text-coral">ARENA</span>
          </h1>
          <div class="flex items-center gap-2">
            <span class="live-indicator flex items-center gap-1.5 text-sm text-gold font-semibold uppercase">
              <span class="w-2 h-2 bg-gold rounded-full animate-pulse"></span>
              Live
            </span>
            <span class="h-4 w-0.5 bg-teal/40 rounded-full"></span>
            <span class="text-white/60 text-sm font-medium">Room: <span class="text-gold font-semibold">{{ route.params.code }}</span></span>
          </div>
        </div>
      </div>

      <!-- Center: Start Button or Game Status -->
      <div class="flex items-center gap-4">
        <button
          v-if="gamePhase === 'waiting'"
          @click="startGame"
          :disabled="!canStartGame"
          :class="canStartGame ? 'btn-start' : 'btn-start-disabled'"
          class="px-8 py-3 rounded-xl font-bold text-lg tracking-wide transition-all flex items-center gap-3"
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

        <div v-if="gamePhase === 'playing'" class="flex items-center gap-8">
          <div class="text-center">
            <div class="text-white/70 text-sm font-semibold uppercase tracking-wider">Players</div>
            <div class="text-4xl font-bold tabular-nums text-teal">{{ leaderboard.length }}</div>
          </div>
          <div class="text-center">
            <div class="text-white/70 text-sm font-semibold uppercase tracking-wider">Accuracy</div>
            <div 
              class="text-4xl font-bold tabular-nums"
              :class="{
                'text-gold': groupAccuracy >= 70,
                'text-orange': groupAccuracy >= 40 && groupAccuracy < 70,
                'text-coral': groupAccuracy < 40
              }"
            >{{ groupAccuracy }}%</div>
          </div>
        </div>

        <div v-if="gamePhase === 'ended'" class="status-ended px-5 py-2.5 rounded-xl flex items-center gap-3">
          <div class="w-3 h-3 bg-coral rounded-full"></div>
          <span class="text-coral font-bold text-lg uppercase tracking-wide">Game Ended</span>
        </div>
      </div>

      <!-- Global Timer -->
      <div 
        class="timer-box flex items-center gap-3 px-5 py-2.5 rounded-xl"
        :class="{
          'timer-normal': gameTimer > 60,
          'timer-warning': gameTimer <= 60 && gameTimer > 0,
          'timer-danger': gameTimer === 0
        }"
      >
        <svg class="w-5 h-5" :class="gameTimer <= 60 ? 'text-coral' : 'text-gold'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <div 
          class="text-3xl font-bold tabular-nums tracking-wider"
          :class="gameTimer <= 60 ? 'text-coral animate-pulse' : 'text-white'"
        >
          {{ formatTime(gameTimer) }}
        </div>
      </div>
    </header>

    <!-- Main Content Area -->
    <div class="flex-1 relative">
      <div id="dashboard-container" class="absolute inset-0 bg-base/50"></div>

      <!-- Floating Leaderboard -->
      <div class="absolute top-4 right-4 w-72">
        <div class="leaderboard-card rounded-lg overflow-hidden">
          <div class="leaderboard-header px-4 py-2">
            <span class="text-teal-dark font-bold text-xl tracking-wide uppercase">Leaderboard</span>
          </div>

          <div class="max-h-72 overflow-y-auto">
            <div
              v-for="player in leaderboard"
              :key="player.id"
              class="leaderboard-item flex items-center gap-3 px-4 py-2 cursor-pointer"
              @click="focusOnPlayer(player.id)"
            >
              <span
                class="w-6 text-xl font-bold tabular-nums shrink-0"
                :class="{
                  'text-gold-dark': player.rank === 1,
                  'text-dark/70': player.rank === 2,
                  'text-orange-dark': player.rank === 3,
                  'text-dark/40': player.rank > 3
                }"
              >{{ player.rank }}</span>

              <div
                class="w-3 h-3 rounded-full shrink-0 border border-white/20"
                :style="{ backgroundColor: player.color }"
              ></div>

              <div class="flex-1 min-w-0">
                <span class="text-dark font-semibold text-lg truncate block">{{ player.name }}</span>
              </div>

              <span class="text-teal-dark font-bold text-xl tabular-nums">{{ player.score }}</span>
            </div>

            <div v-if="leaderboard.length === 0" class="px-4 py-6 text-center">
              <div class="text-dark/60 font-semibold text-lg">Waiting for players...</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Spectator Badge -->
      <div class="spectator-badge absolute bottom-4 left-4 px-5 py-2.5 rounded-full flex items-center gap-2">
        <svg class="w-5 h-5 text-teal-dark" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
        </svg>
        <span class="text-teal-dark text-sm font-bold tracking-wide uppercase">Spectator Mode</span>
      </div>

      <!-- Prominent Game Code Display -->
      <Transition name="code-fade">
        <div 
          v-if="gamePhase === 'waiting'" 
          class="absolute bottom-8 left-1/2 -translate-x-1/2 z-20"
        >
          <div class="game-code-card rounded-2xl p-8 text-center">
            <div class="text-dark/60 text-lg font-semibold uppercase tracking-widest mb-3">Join with Code</div>
            <div class="game-code-display text-7xl font-extrabold tracking-[0.3em] text-dark select-all cursor-pointer" @click="copyCode">
              {{ route.params.code }}
            </div>
            <div class="mt-4 flex items-center justify-center gap-2 text-dark/50">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
              </svg>
              <span class="text-sm font-medium">Click to copy</span>
            </div>
          </div>
        </div>
      </Transition>

      <!-- Instructions -->
      <Transition name="instructions-fade">
        <div 
          v-if="gamePhase === 'waiting' && showInstructions" 
          class="absolute top-6 left-1/2 -translate-x-1/2 z-20"
        >
          <div class="instructions-card rounded-2xl p-8 max-w-xl">
            <div class="flex items-center justify-between mb-8">
              <h2 class="text-3xl font-bold uppercase tracking-wider">
                <span class="text-teal-dark">How to Play</span>
              </h2>
              <button 
                @click="showInstructions = false"
                class="close-btn w-9 h-9 rounded-full flex items-center justify-center transition-colors"
              >
                <svg class="w-5 h-5 text-white/50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div class="space-y-5 text-xl">
              <p class="text-dark/80">
                <span class="text-teal-dark font-bold">Movement:</span> 
                Use <kbd class="key">↑</kbd><kbd class="key">↓</kbd><kbd class="key">←</kbd><kbd class="key">→</kbd> or <kbd class="key">W</kbd><kbd class="key">A</kbd><kbd class="key">S</kbd><kbd class="key">D</kbd> to move.
              </p>
              <p class="text-dark/80">
                <span class="text-coral-dark font-bold">Shooting: </span> 
                <kbd class="key">Left Click</kbd> or <kbd class="key">Space</kbd> to shoot.
              </p>
              <p class="text-dark/80">
                <span class="text-orange-dark font-bold">Weapons:</span> 
                Press <kbd class="key">1</kbd><kbd class="key">2</kbd><kbd class="key">3</kbd> to switch weapons.
              </p>
              <p class="text-dark/80">
                <span class="text-gold-dark font-bold">Reload & Respawn:</span> 
                Answer <span class="text-teal-dark font-bold">questions</span> correctly.
              </p>
            </div>

            <p class="text-dark/50 text-base text-center mt-8">Click <span class="text-coral-dark">✕</span> to dismiss</p>
          </div>
        </div>
      </Transition>

      <!-- Game Ended Overlay -->
      <div v-if="gamePhase === 'ended'" class="absolute inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center z-50">
        <div class="game-over-card rounded-3xl p-8 max-w-lg w-full mx-4">
          <div class="text-center mb-6">
            <div class="trophy-icon w-20 h-20 mx-auto mb-4 rounded-full flex items-center justify-center">
              <svg class="w-10 h-10 text-dark" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
              </svg>
            </div>
            <h2 class="text-4xl font-bold text-dark mb-2 uppercase tracking-wider">Game Over!</h2>
            <p class="text-dark/60 text-lg">Final Standings</p>
          </div>

          <div class="space-y-3 mb-6">
            <div
              v-for="(player, index) in leaderboard.slice(0, 3)"
              :key="player.id"
              class="podium-item flex items-center gap-4 p-4 rounded-xl"
              :class="{
                'podium-first': index === 0,
                'podium-second': index === 1,
                'podium-third': index === 2
              }"
            >
              <div
                class="rank-badge w-10 h-10 rounded-full flex items-center justify-center text-lg font-bold shrink-0"
                :class="{
                  'rank-first': index === 0,
                  'rank-second': index === 1,
                  'rank-third': index === 2
                }"
              >
                {{ index + 1 }}
              </div>

              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <div
                    class="w-4 h-4 rounded-full shrink-0 border border-dark/20"
                    :style="{ backgroundColor: player.color }"
                  ></div>
                  <span class="text-dark font-bold text-lg truncate">{{ player.name }}</span>
                </div>
              </div>

              <div class="text-right">
                <span class="text-2xl font-bold" :class="{
                  'text-gold-dark': index === 0,
                  'text-dark/70': index === 1,
                  'text-orange-dark': index === 2
                }">{{ player.score.toLocaleString() }}</span>
                <span class="text-dark/50 text-sm ml-1">pts</span>
              </div>
            </div>
          </div>

          <button
            @click="router.push('/')"
            class="btn-home w-full py-4 font-bold text-lg rounded-xl transition-all"
          >
            Return to Home
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700;800&display=swap');

.dashboard {
  font-family: 'Outfit', sans-serif;
}

/* Color Classes */
.bg-base { background-color: #0d1117; }
.bg-teal { background-color: #5A9CB5; }
.bg-gold { background-color: #FACE68; }
.bg-orange { background-color: #FAAC68; }
.bg-coral { background-color: #FA6868; }

.text-dark { color: #2c3e50; }
.text-teal { color: #7BB8CC; }
.text-teal-dark { color: #4a8a9f; }
.text-gold { color: #FFD980; }
.text-gold-dark { color: #c49a30; }
.text-orange { color: #FFBE7A; }
.text-orange-dark { color: #d48a30; }
.text-coral { color: #FF7A7A; }
.text-coral-dark { color: #d94545; }

/* Header */
.header-bar {
  background: linear-gradient(180deg, rgba(22, 27, 34, 0.98) 0%, rgba(13, 17, 23, 0.95) 100%);
  backdrop-filter: blur(20px);
  border-bottom: 1px solid rgba(90, 156, 181, 0.2);
}

/* Logo */
.logo-icon {
  background: linear-gradient(135deg, #FACE68 0%, #FA6868 100%);
  border: 2px solid rgba(250, 206, 104, 0.5);
  box-shadow: 0 4px 15px rgba(250, 104, 104, 0.3);
}

/* Buttons */
.btn-start {
  background: linear-gradient(135deg, #7BB8CC 0%, #5A9CB5 100%);
  color: #0d1117;
  border: 2px solid rgba(123, 184, 204, 0.6);
  box-shadow: 0 10px 30px rgba(123, 184, 204, 0.4);
  font-weight: 800;
}

.btn-start:hover {
  transform: translateY(-2px);
  box-shadow: 0 15px 40px rgba(123, 184, 204, 0.5);
  background: linear-gradient(135deg, #8EC9DB 0%, #6AADBD 100%);
}

.btn-start-disabled {
  background: linear-gradient(135deg, rgba(90, 156, 181, 0.3) 0%, rgba(74, 138, 159, 0.3) 100%);
  color: rgba(255, 255, 255, 0.7);
  border: 2px solid rgba(90, 156, 181, 0.4);
  cursor: not-allowed;
}

.btn-danger {
  background: linear-gradient(135deg, #FA6868 0%, #d94545 100%);
  color: white;
  border: 1px solid rgba(250, 104, 104, 0.5);
}

.btn-home {
  background: linear-gradient(135deg, #7BB8CC 0%, #5A9CB5 100%);
  color: #0d1117;
  border: 2px solid rgba(123, 184, 204, 0.5);
  box-shadow: 0 10px 30px rgba(123, 184, 204, 0.3);
  font-weight: 800;
}

.btn-home:hover {
  transform: translateY(-2px);
  box-shadow: 0 15px 40px rgba(123, 184, 204, 0.4);
  background: linear-gradient(135deg, #8EC9DB 0%, #6AADBD 100%);
}

/* Status boxes */
.status-ended {
  background: rgba(250, 104, 104, 0.1);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(250, 104, 104, 0.3);
}

/* Timer */
.timer-box {
  backdrop-filter: blur(10px);
  border: 1px solid;
}

.timer-normal {
  background: rgba(22, 27, 34, 0.8);
  border-color: rgba(90, 156, 181, 0.3);
}

.timer-warning {
  background: rgba(250, 104, 104, 0.15);
  border-color: rgba(250, 104, 104, 0.4);
}

.timer-danger {
  background: rgba(250, 104, 104, 0.25);
  border-color: rgba(250, 104, 104, 0.6);
}

/* Cards - Light cream interior */
.card-modal {
  background: linear-gradient(145deg, #f5f0e6 0%, #ebe5d9 100%);
  border: 1px solid rgba(200, 180, 150, 0.5);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.4);
}

.leaderboard-card {
  background: linear-gradient(145deg, #f5f0e6 0%, #ebe5d9 100%);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(200, 180, 150, 0.5);
}

.leaderboard-header {
  background: rgba(90, 156, 181, 0.15);
  border-bottom: 1px solid rgba(200, 180, 150, 0.4);
}

.leaderboard-item {
  transition: background 0.2s ease;
}

.leaderboard-item:hover {
  background: rgba(90, 156, 181, 0.15);
}

.spectator-badge {
  background: linear-gradient(145deg, #f5f0e6 0%, #ebe5d9 100%);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(200, 180, 150, 0.5);
}

.instructions-card {
  background: linear-gradient(145deg, #f5f0e6 0%, #ebe5d9 100%);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(200, 180, 150, 0.5);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.4);
}

.close-btn {
  background: rgba(90, 156, 181, 0.2);
  border: 1px solid rgba(90, 156, 181, 0.3);
}

.close-btn:hover {
  background: rgba(250, 104, 104, 0.3);
}

.game-over-card {
  background: linear-gradient(145deg, #f5f0e6 0%, #ebe5d9 100%);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(200, 180, 150, 0.5);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.4);
}

.trophy-icon {
  background: linear-gradient(135deg, #FACE68 0%, #FA6868 100%);
  border: 3px solid rgba(250, 206, 104, 0.5);
  box-shadow: 0 10px 30px rgba(250, 104, 104, 0.4);
}

/* Podium items */
.podium-first {
  background: linear-gradient(135deg, rgba(196, 154, 48, 0.2) 0%, rgba(196, 154, 48, 0.08) 100%);
  border: 1px solid rgba(196, 154, 48, 0.4);
}

.podium-second {
  background: linear-gradient(135deg, rgba(90, 156, 181, 0.15) 0%, rgba(90, 156, 181, 0.05) 100%);
  border: 1px solid rgba(90, 156, 181, 0.3);
}

.podium-third {
  background: linear-gradient(135deg, rgba(212, 138, 48, 0.2) 0%, rgba(212, 138, 48, 0.08) 100%);
  border: 1px solid rgba(212, 138, 48, 0.4);
}

.rank-first {
  background: linear-gradient(135deg, #FACE68 0%, #FAAC68 100%);
  color: #0d1117;
  border: 2px solid #FACE68;
}

.rank-second {
  background: linear-gradient(135deg, #e0e0e0 0%, #bdbdbd 100%);
  color: #0d1117;
  border: 2px solid #e0e0e0;
}

.rank-third {
  background: linear-gradient(135deg, #FAAC68 0%, #e09550 100%);
  color: #0d1117;
  border: 2px solid #FAAC68;
}

/* Live indicator */
.live-indicator span {
  animation: live-pulse 2s ease-in-out infinite;
}

@keyframes live-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(0.9); }
}

/* Scrollbar */
.max-h-72::-webkit-scrollbar {
  width: 6px;
}

.max-h-72::-webkit-scrollbar-track {
  background: rgba(13, 17, 23, 0.5);
  border-radius: 3px;
}

.max-h-72::-webkit-scrollbar-thumb {
  background: rgba(90, 156, 181, 0.3);
  border-radius: 3px;
}

.max-h-72::-webkit-scrollbar-thumb:hover {
  background: rgba(90, 156, 181, 0.5);
}

/* Transitions */
.instructions-fade-enter-active,
.instructions-fade-leave-active {
  transition: all 0.4s ease;
}

.instructions-fade-enter-from,
.instructions-fade-leave-to {
  opacity: 0;
  transform: scale(0.95);
}

.code-fade-enter-active,
.code-fade-leave-active {
  transition: all 0.4s ease;
}

.code-fade-enter-from,
.code-fade-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(20px);
}

/* Game Code Card */
.game-code-card {
  background: linear-gradient(145deg, #f5f0e6 0%, #ebe5d9 100%);
  backdrop-filter: blur(20px);
  border: 2px solid rgba(90, 156, 181, 0.4);
  box-shadow: 
    0 25px 50px -12px rgba(0, 0, 0, 0.4),
    0 0 80px rgba(90, 156, 181, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.5);
}

.game-code-display {
  background: linear-gradient(135deg, #4a8a9f 0%, #5A9CB5 50%, #4a8a9f 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  text-shadow: none;
  filter: drop-shadow(0 2px 4px rgba(90, 156, 181, 0.3));
  animation: code-glow 2s ease-in-out infinite alternate;
}

.game-code-display:hover {
  transform: scale(1.02);
  transition: transform 0.2s ease;
}

@keyframes code-glow {
  0% {
    filter: drop-shadow(0 2px 8px rgba(90, 156, 181, 0.3));
  }
  100% {
    filter: drop-shadow(0 4px 20px rgba(90, 156, 181, 0.5));
  }
}

/* Keyboard keys */
.key {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.75rem;
  height: 1.75rem;
  padding: 0 0.5rem;
  margin: 0 0.125rem;
  background: #5A9CB5;
  border: 1px solid #4a8a9f;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 700;
  color: white;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.15);
  text-transform: uppercase;
  letter-spacing: 0.025em;
  font-family: inherit;
  vertical-align: middle;
}

h1 {
  font-weight: 800;
  letter-spacing: 0.1em;
}
</style>
