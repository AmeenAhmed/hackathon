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

        // Send immediately - messages are queued until connection opens
        // console.log('Sending rejoin request:', { code, playerId });
        ws.send('rejoinRoom', {
          code: code as string,
          playerId: playerId as string
        });
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
      <div class="loading-card">
        <div class="logo-container">
          <div class="logo-icon">
            <span>W</span>
          </div>
          <h1 class="logo-text">
            <span class="way">WAY</span>
            <span class="arena">ARENA</span>
          </h1>
        </div>
        <h2>Connecting to game...</h2>
        <div class="spinner"></div>
        <p class="loading-hint">Preparing your arena experience</p>
      </div>
    </div>

    <div v-else-if="error" class="error">
      <div class="error-card">
        <div class="error-icon">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <h2>Error</h2>
        <p>{{ error }}</p>
        <button @click="goHome">Return Home</button>
      </div>
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
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700;800&display=swap');

.game-page {
  width: 100vw;
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background-color: #0d1117;
  overflow: hidden;
  font-family: 'Outfit', sans-serif;
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
  cursor: url('/assets/images/pointer-big.png'), crosshair;
}

/* Loading Screen */
.loading {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0d1117;
}

.loading-card {
  text-align: center;
  padding: 3rem;
  background: linear-gradient(145deg, rgba(22, 27, 34, 0.95) 0%, rgba(13, 17, 23, 0.98) 100%);
  border: 1px solid rgba(90, 156, 181, 0.3);
  border-radius: 1.5rem;
  box-shadow: 0 25px 50px rgba(0, 0, 0, 0.5);
  min-width: 320px;
}

.logo-container {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  margin-bottom: 2rem;
}

.logo-icon {
  width: 50px;
  height: 50px;
  background: linear-gradient(135deg, #FACE68 0%, #FA6868 100%);
  border: 2px solid rgba(250, 206, 104, 0.5);
  border-radius: 0.75rem;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 20px rgba(250, 104, 104, 0.3);
}

.logo-icon span {
  font-size: 1.75rem;
  font-weight: 800;
  color: #0d1117;
}

.logo-text {
  display: flex;
  flex-direction: column;
  text-align: left;
  line-height: 1.1;
}

.logo-text .way {
  font-size: 1.5rem;
  font-weight: 800;
  color: white;
  letter-spacing: 0.1em;
}

.logo-text .arena {
  font-size: 1.5rem;
  font-weight: 800;
  color: #FF7A7A;
  letter-spacing: 0.1em;
}

.loading h2 {
  color: #7BB8CC;
  font-weight: 700;
  font-size: 1.25rem;
  margin-bottom: 1.5rem;
}

.spinner {
  border: 3px solid rgba(90, 156, 181, 0.2);
  border-top: 3px solid #5A9CB5;
  border-radius: 50%;
  width: 50px;
  height: 50px;
  animation: spin 1s linear infinite;
  margin: 0 auto 1.5rem;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.loading-hint {
  color: rgba(255, 255, 255, 0.5);
  font-size: 0.875rem;
}

/* Error Screen */
.error {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0d1117;
}

.error-card {
  text-align: center;
  padding: 3rem;
  background: linear-gradient(145deg, rgba(22, 27, 34, 0.95) 0%, rgba(13, 17, 23, 0.98) 100%);
  border: 1px solid rgba(250, 104, 104, 0.3);
  border-radius: 1.5rem;
  box-shadow: 0 25px 50px rgba(0, 0, 0, 0.5);
  min-width: 320px;
  max-width: 400px;
}

.error-icon {
  width: 60px;
  height: 60px;
  margin: 0 auto 1.5rem;
  background: rgba(250, 104, 104, 0.15);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.error-icon svg {
  width: 32px;
  height: 32px;
  color: #FF7A7A;
}

.error h2 {
  color: #FF7A7A;
  font-weight: 700;
  font-size: 1.5rem;
  margin-bottom: 0.75rem;
}

.error p {
  color: rgba(255, 255, 255, 0.7);
  margin-bottom: 1.5rem;
  line-height: 1.5;
}

.error button {
  padding: 12px 28px;
  background: linear-gradient(135deg, #5A9CB5 0%, #4a8a9f 100%);
  color: white;
  border: 2px solid rgba(90, 156, 181, 0.5);
  border-radius: 10px;
  cursor: pointer;
  font-size: 16px;
  font-weight: 700;
  transition: all 0.3s ease;
}

.error button:hover {
  transform: translateY(-2px);
  box-shadow: 0 10px 30px rgba(90, 156, 181, 0.3);
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
  background: linear-gradient(145deg, rgba(22, 27, 34, 0.98) 0%, rgba(13, 17, 23, 0.99) 100%);
  border: 2px solid rgba(90, 156, 181, 0.4);
  border-radius: 20px;
  padding: 40px;
  text-align: center;
  max-width: 400px;
  box-shadow: 0 25px 50px rgba(0, 0, 0, 0.6);
}

.trophy-icon {
  width: 80px;
  height: 80px;
  margin: 0 auto 20px;
  background: linear-gradient(135deg, #FACE68 0%, #FA6868 100%);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 3px solid rgba(250, 206, 104, 0.5);
  box-shadow: 0 10px 30px rgba(250, 104, 104, 0.3);
}

.trophy-icon svg {
  width: 40px;
  height: 40px;
  color: #0d1117;
}

.game-ended-modal h2 {
  font-size: 32px;
  color: #FFD980;
  margin-bottom: 12px;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 2px;
}

.game-ended-modal p {
  color: rgba(255, 255, 255, 0.7);
  font-size: 16px;
  margin-bottom: 24px;
}

.home-button {
  background: linear-gradient(135deg, #5A9CB5 0%, #4a8a9f 100%);
  color: white;
  border: 2px solid rgba(90, 156, 181, 0.5);
  padding: 14px 32px;
  border-radius: 10px;
  font-size: 16px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.3s ease;
  text-transform: uppercase;
  letter-spacing: 1px;
  box-shadow: 0 10px 30px rgba(90, 156, 181, 0.3);
}

.home-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 15px 40px rgba(90, 156, 181, 0.4);
  background: linear-gradient(135deg, #6AADBD 0%, #5A9CB5 100%);
}
</style>