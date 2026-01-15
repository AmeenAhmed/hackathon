<script setup lang="ts">
import { onMounted, onUnmounted, ref, nextTick } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { usePlayerStore } from '../stores/playerStore';
import { useWS } from '../composables/useWS';
import GameManager from '../game/GameManager';
import Quiz from '../components/Quiz.vue';

const route = useRoute();
const router = useRouter();
const playerStore = usePlayerStore();
const ws = useWS();
const gameManager = ref<GameManager | null>(null);
const isLoading = ref(true);
const error = ref<string | null>(null);
const quizRef = ref<any>(null);
const isGameEnded = ref(false);

const goHome = () => {
  if (router) {
    router.push('/');
  } else {
    window.location.href = '/';
  }
};

onMounted(async () => {
  const code = route.params.code as string;
  const playerId = route.params.playerId as string;

  // Validate route params
  if (!code || !playerId) {
    error.value = 'Invalid game URL. Missing room code or player ID.';
    isLoading.value = false;
    return;
  }

  try {
    // Always initialize WebSocket
    ws.init();

    // If we don't have a player ID, wait for rejoin confirmation
    if (!playerStore.id || playerStore.id !== playerId) {
      await new Promise<void>((resolve, reject) => {
        const timeout = setTimeout(() => {
          // Don't reject, just resolve to continue with game initialization
          // console.warn('WebSocket connection timeout, continuing anyway');
          resolve();
        }, 3000);

        ws.on('rejoinedRoom', (message: any) => {
          clearTimeout(timeout);
          // console.log('Rejoined room:', message);
          playerStore.setPlayerData(message.player);
          // Store terrain data for the game to use
          if (message.mapData) {
            sessionStorage.setItem('mapData', JSON.stringify(message.mapData));
          }
          resolve();
        });

        ws.on('error', (message: any) => {
          clearTimeout(timeout);
          // console.error('WebSocket error:', message);
          // Don't reject, continue with game initialization
          resolve();
        });

        setTimeout(() => {
          // console.log('Sending rejoin request:', { code, playerId });
          ws.send('rejoinRoom', {
            code: code as string,
            playerId: playerId as string
          });
        }, 200);
      });
    }

    // Make player store and quiz control available to Phaser
    (window as any).playerStore = playerStore;
    (window as any).showDeathQuiz = () => {
      if (quizRef.value) {
        quizRef.value.startQuiz('death');
      }
    };
    (window as any).showAmmoQuiz = () => {
      if (quizRef.value) {
        quizRef.value.startQuiz('ammo');
      }
    };

    // Set loading to false to render the game container
    isLoading.value = false;

    // Wait for DOM to be updated before initializing Phaser
    await nextTick();

    // Small delay to ensure container is fully ready
    await new Promise(resolve => setTimeout(resolve, 100));

    // Initialize Phaser game AFTER the container is rendered
    // console.log('Initializing Phaser game with:', { code, playerId });
    const container = document.getElementById('game-container');
    // console.log('Container element before init:', container);

    gameManager.value = new GameManager();
    gameManager.value.init(
      code as string,
      playerId as string,
      ws
    );
    // Make gameManager globally available after it's created
    (window as any).gameManager = gameManager.value;
    // console.log('Phaser game initialized');

    // Setup WebSocket listeners for game updates
    ws.on('gameUpdate', (data: any) => {
      if (gameManager.value) {
        gameManager.value.handleGameUpdate(data.gameState);
      }
    });

    ws.on('initialState', (data: any) => {
      if (gameManager.value) {
        gameManager.value.handleGameUpdate(data.gameState);
      }
    });

    ws.on('gameStarted', (data: any) => {
      // console.log('GamePage received gameStarted event');
      if (gameManager.value) {
        // Update the game state to reflect that the game has started
        gameManager.value.handleGameUpdate({ gamePhase: 'playing' });
      }
    });

    ws.on('gameEnded', (data: any) => {
      // console.log('GamePage received gameEnded event');
      isGameEnded.value = true;
      if (gameManager.value) {
        gameManager.value.handleGameUpdate({ gamePhase: 'ended' });
      }
    });
  } catch (err) {
    // console.error('Game initialization error:', err);
    if (err instanceof Error) {
      error.value = err.message;
    } else if (typeof err === 'string') {
      error.value = err;
    } else {
      error.value = 'Failed to initialize game';
    }
    isLoading.value = false;
  }
});

onUnmounted(() => {
  // Clean up Phaser game
  if (gameManager.value) {
    gameManager.value.destroy();
    gameManager.value = null;
  }

  // Clean up window references
  if ((window as any).playerStore) {
    delete (window as any).playerStore;
  }
  if ((window as any).showDeathQuiz) {
    delete (window as any).showDeathQuiz;
  }
  if ((window as any).showAmmoQuiz) {
    delete (window as any).showAmmoQuiz;
  }
  if ((window as any).onQuizComplete) {
    delete (window as any).onQuizComplete;
  }
  if ((window as any).gameManager) {
    delete (window as any).gameManager;
  }
});
</script>

<template>
  <div class="game-page">
    <div v-if="isLoading" class="loading">
      <h2>Connecting to game...</h2>
      <div class="spinner"></div>
    </div>

    <div v-else-if="error" class="error">
      <h2>Error</h2>
      <p>{{ error }}</p>
      <button @click="goHome">Return Home</button>
    </div>

    <div v-else id="game-container" class="game-container"></div>

    <!-- Unified Quiz Overlay -->
    <Quiz ref="quizRef" />

    <!-- Game Ended Overlay -->
    <div v-if="isGameEnded" class="game-ended-overlay">
      <div class="game-ended-modal">
        <div class="trophy-icon">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">
            <path d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </div>
        <h2>Game Over!</h2>
        <p>Time's up! Check the dashboard for final results.</p>
        <button @click="goHome" class="home-button">Return to Home</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.game-page {
  width: 100vw;
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background-color: #1a1a1a;
  overflow: hidden;
}

.game-container {
  width: 100%;
  height: 100%;
}

/* Ensure pixel-perfect rendering */
.game-container canvas {
  image-rendering: pixelated;
  image-rendering: -moz-crisp-edges;
  image-rendering: crisp-edges;
}

.loading, .error {
  text-align: center;
  color: white;
}

.spinner {
  border: 3px solid #f3f3f3;
  border-top: 3px solid #3498db;
  border-radius: 50%;
  width: 40px;
  height: 40px;
  animation: spin 1s linear infinite;
  margin: 20px auto;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.error button {
  margin-top: 20px;
  padding: 10px 20px;
  background-color: #3498db;
  color: white;
  border: none;
  border-radius: 5px;
  cursor: pointer;
  font-size: 16px;
}

.error button:hover {
  background-color: #2980b9;
}

/* Game Ended Overlay */
.game-ended-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(8px);
}

.game-ended-modal {
  background: linear-gradient(135deg, #1e1e2e 0%, #2d2d44 100%);
  border: 2px solid #fbbf24;
  border-radius: 20px;
  padding: 40px;
  text-align: center;
  max-width: 400px;
  box-shadow: 0 0 40px rgba(251, 191, 36, 0.3);
}

.trophy-icon {
  width: 80px;
  height: 80px;
  margin: 0 auto 20px;
  background: linear-gradient(135deg, #fbbf24 0%, #f59e0b 100%);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.trophy-icon svg {
  width: 40px;
  height: 40px;
  color: #1e1e2e;
}

.game-ended-modal h2 {
  font-size: 32px;
  color: #fbbf24;
  margin-bottom: 12px;
  font-weight: bold;
  text-transform: uppercase;
  letter-spacing: 2px;
}

.game-ended-modal p {
  color: #a1a1aa;
  font-size: 16px;
  margin-bottom: 24px;
}

.home-button {
  background: linear-gradient(135deg, #8b5cf6 0%, #d946ef 100%);
  color: white;
  border: none;
  padding: 14px 32px;
  border-radius: 10px;
  font-size: 16px;
  font-weight: bold;
  cursor: pointer;
  transition: all 0.2s ease;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.home-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(139, 92, 246, 0.4);
}
</style>