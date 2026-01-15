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
</style>