<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { useWS } from '../composables/useWS';
import { useRouter } from 'vue-router';
import { usePlayerStore } from '../stores/playerStore';

const { init, send, on, off } = useWS();
const router = useRouter();
const playerStore = usePlayerStore();

const code = ref('');
const name = ref('');
const error = ref('');
const isLoading = ref(false);

const isValidCode = computed(() => {
  return !!code.value && !!name.value && code.value.length === 6;
});

function clearError() {
  error.value = '';
}

function handleError(message: any) {
  isLoading.value = false;
  error.value = message.error || 'An unexpected error occurred';
}

function handleRoomCreated(message: any) {
  isLoading.value = false;
  router.push(`/dashboard/${message.roomCode}`);
}

function handleJoinedRoom(message: any) {
  isLoading.value = false;
  playerStore.setPlayerData(message.player);
  if (message.mapData) {
    sessionStorage.setItem('mapData', JSON.stringify(message.mapData));
  }
  router.push(`/game/${code.value}/${message.playerId}`);
}

function createRoom() {
  clearError();
  isLoading.value = true;
  send('createRoom');
}

function joinRoom() {
  clearError();
  isLoading.value = true;
  send('joinRoom', { code: code.value, name: name.value });
}

onMounted(() => {
  init();
  on('error', handleError);
  on('roomCreated', handleRoomCreated);
  on('joinedRoom', handleJoinedRoom);
});

onUnmounted(() => {
  off('error', handleError);
  off('roomCreated', handleRoomCreated);
  off('joinedRoom', handleJoinedRoom);
});
</script>

<template>
  <div class="home-page w-screen h-screen flex justify-center items-center relative overflow-hidden">
    <!-- Desert Sky Gradient -->
    <div class="fixed inset-0 desert-sky"></div>
    
    <!-- Sun Glow -->
    <div class="fixed inset-0 pointer-events-none overflow-hidden">
      <div class="sun"></div>
      <div class="sun-glow"></div>
    </div>

    <!-- Desert Mountains/Dunes - Back Layer -->
    <div class="fixed bottom-0 left-0 right-0 pointer-events-none">
      <svg class="w-full h-64" viewBox="0 0 1440 256" preserveAspectRatio="none">
        <!-- Far mountains -->
        <path d="M0,256 L0,180 L100,140 L200,160 L350,100 L500,150 L650,90 L800,140 L950,110 L1100,150 L1250,120 L1440,160 L1440,256 Z" fill="#4a3728" opacity="0.5"/>
        <!-- Mid mountains -->
        <path d="M0,256 L0,190 L150,150 L300,180 L450,130 L600,170 L750,120 L900,160 L1050,140 L1200,170 L1350,150 L1440,180 L1440,256 Z" fill="#5c4332" opacity="0.6"/>
        <!-- Near dunes -->
        <path d="M0,256 L0,210 L120,185 L280,200 L400,175 L550,195 L700,170 L850,190 L1000,180 L1150,200 L1300,185 L1440,205 L1440,256 Z" fill="#6b4f3a" opacity="0.7"/>
      </svg>
    </div>

    <!-- Sand Ground -->
    <div class="fixed bottom-0 left-0 right-0 h-28 sand-ground pointer-events-none"></div>

    <!-- Cactus Left Side -->
    <div class="fixed bottom-24 left-[8%] pointer-events-none cactus-shadow">
      <svg width="80" height="160" viewBox="0 0 80 160" class="cactus">
        <!-- Main stem -->
        <rect x="32" y="20" width="16" height="140" rx="8" fill="#2d5a3d"/>
        <rect x="34" y="22" width="4" height="136" rx="2" fill="#3d7a52" opacity="0.5"/>
        <!-- Left arm -->
        <rect x="8" y="50" width="32" height="12" rx="6" fill="#2d5a3d"/>
        <rect x="8" y="38" width="12" height="50" rx="6" fill="#2d5a3d"/>
        <rect x="10" y="40" width="3" height="46" rx="1.5" fill="#3d7a52" opacity="0.5"/>
        <!-- Right arm -->
        <rect x="40" y="70" width="28" height="12" rx="6" fill="#2d5a3d"/>
        <rect x="56" y="55" width="12" height="45" rx="6" fill="#2d5a3d"/>
        <rect x="58" y="57" width="3" height="41" rx="1.5" fill="#3d7a52" opacity="0.5"/>
      </svg>
    </div>

    <!-- Cactus Right Side -->
    <div class="fixed bottom-20 right-[10%] pointer-events-none cactus-shadow">
      <svg width="60" height="120" viewBox="0 0 60 120" class="cactus">
        <!-- Main stem -->
        <rect x="22" y="15" width="16" height="105" rx="8" fill="#2d5a3d"/>
        <rect x="24" y="17" width="4" height="101" rx="2" fill="#3d7a52" opacity="0.5"/>
        <!-- Left arm -->
        <rect x="2" y="40" width="28" height="10" rx="5" fill="#2d5a3d"/>
        <rect x="2" y="30" width="10" height="35" rx="5" fill="#2d5a3d"/>
        <rect x="4" y="32" width="2" height="31" rx="1" fill="#3d7a52" opacity="0.5"/>
        <!-- Right arm -->
        <rect x="30" y="55" width="24" height="10" rx="5" fill="#2d5a3d"/>
        <rect x="44" y="45" width="10" height="30" rx="5" fill="#2d5a3d"/>
        <rect x="46" y="47" width="2" height="26" rx="1" fill="#3d7a52" opacity="0.5"/>
      </svg>
    </div>

    <!-- Small Cactus -->
    <div class="fixed bottom-24 left-[25%] pointer-events-none cactus-shadow">
      <svg width="30" height="50" viewBox="0 0 30 50" class="cactus-small">
        <rect x="10" y="8" width="10" height="42" rx="5" fill="#2d5a3d"/>
        <rect x="12" y="10" width="2" height="38" rx="1" fill="#3d7a52" opacity="0.5"/>
      </svg>
    </div>

    <!-- Another Small Cactus -->
    <div class="fixed bottom-28 right-[22%] pointer-events-none cactus-shadow">
      <svg width="35" height="55" viewBox="0 0 35 55" class="cactus-small">
        <rect x="12" y="10" width="11" height="45" rx="5.5" fill="#2d5a3d"/>
        <rect x="14" y="12" width="2" height="41" rx="1" fill="#3d7a52" opacity="0.5"/>
        <rect x="0" y="25" width="16" height="8" rx="4" fill="#2d5a3d"/>
        <rect x="0" y="18" width="8" height="20" rx="4" fill="#2d5a3d"/>
      </svg>
    </div>

    <!-- Tumbleweeds -->
    <div class="tumbleweed tumbleweed-1"></div>
    <div class="tumbleweed tumbleweed-2"></div>
    <div class="tumbleweed tumbleweed-3"></div>

    <!-- Dust particles -->
    <div class="dust-container fixed inset-0 pointer-events-none overflow-hidden">
      <div class="dust dust-1"></div>
      <div class="dust dust-2"></div>
      <div class="dust dust-3"></div>
      <div class="dust dust-4"></div>
      <div class="dust dust-5"></div>
    </div>

    <div class="flex gap-8 relative z-10">
      <!-- Main Card -->
      <div class="card-main p-12 rounded-2xl flex flex-col gap-8 min-w-120 items-center">
        <!-- Logo & Title -->
        <div class="flex items-center gap-4">
          <div class="logo-icon w-14 h-14 rounded-xl flex items-center justify-center">
            <span class="text-dark text-3xl font-bold">W</span>
          </div>
          <h1 class="text-4xl font-bold tracking-wide uppercase">
            <span class="text-dark">WAY</span>
            <span class="text-coral-dark">ARENA</span>
          </h1>
        </div>

        <!-- Tagline -->
        <div class="text-center -mt-2">
          <span class="text-teal-dark text-sm font-bold tracking-widest uppercase">Kill to Learn. Answer to Live.</span>
        </div>
        
        <!-- Error Message -->
        <div 
          v-if="error" 
          class="w-full bg-red-50 border border-red-300 text-red-700 px-4 py-3 rounded-lg text-center text-sm font-semibold"
        >
          {{ error }}
        </div>

        <input 
          class="input-field py-3 px-6 font-bold rounded-xl outline-none text-center w-full uppercase"
          :class="error ? 'border-coral' : ''"
          v-model="code"
          name="code" 
          id="code"
          placeholder="Join Code"
          @input="clearError"
        />
        <input 
          class="input-field py-3 px-6 font-bold rounded-xl outline-none text-center w-full"
          v-model="name"
          name="name" 
          id="name"
          placeholder="Your Name"
          @input="clearError"
        />
        <button 
          class="btn-primary px-8 py-3 rounded-xl font-bold cursor-pointer w-full transition-all"
          :class="{
            'opacity-100': isValidCode && !isLoading,
            'opacity-50 pointer-events-none': !isValidCode || isLoading
          }"
          @click="joinRoom"
          :disabled="!isValidCode || isLoading"
        >
          {{ isLoading ? 'JOINING...' : 'JOIN GAME' }}
        </button>
        <div class="w-full border-b-2 border-dashed" style="border-color: rgba(90, 156, 181, 0.4);"></div>
        <button 
          class="btn-secondary px-8 py-3 rounded-xl font-bold cursor-pointer w-full transition-all"
          :class="{ 'opacity-50 pointer-events-none': isLoading }"
          @click="createRoom"
          :disabled="isLoading"
        >
          {{ isLoading ? 'CREATING...' : 'CREATE ROOM' }}
        </button>
      </div>

      <!-- Instructions Panel -->
      <div class="card-secondary p-8 rounded-2xl w-80 flex flex-col gap-6">
        <h2 class="text-xl font-bold uppercase tracking-wider text-center text-teal-dark">
          How to Play
        </h2>

        <!-- Movement -->
        <div class="instruction-block">
          <div class="flex items-center gap-2 mb-3">
            <div class="icon-box bg-teal">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-dark" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
              </svg>
            </div>
            <span class="font-bold text-teal-dark uppercase text-sm tracking-wide">Movement</span>
          </div>
          <div class="flex flex-col gap-2">
              <div class="keys-row">
                <span class="key">↑</span>
                <span class="key">↓</span>
                <span class="key">←</span>
                <span class="key">→</span>
              </div>
            <span class="text-dark/50 text-xs">or</span>
            <div class="keys-row">
              <span class="key">W</span>
              <span class="key">A</span>
              <span class="key">S</span>
              <span class="key">D</span>
            </div>
          </div>
        </div>

        <!-- Shooting -->
        <div class="instruction-block">
          <div class="flex items-center gap-2 mb-3">
            <div class="icon-box bg-coral">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <span class="font-bold text-coral-dark uppercase text-sm tracking-wide">Shooting</span>
          </div>
          <div class="flex items-center gap-2 flex-wrap">
            <span class="key key-wide">Left Click</span>
            <span class="text-dark/50 text-xs">or</span>
            <span class="key key-wide">Space</span>
          </div>
        </div>

        <!-- Weapons -->
        <div class="instruction-block">
          <div class="flex items-center gap-2 mb-3">
            <div class="icon-box bg-orange">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-dark" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16m-7 6h7" />
              </svg>
            </div>
            <span class="font-bold text-orange-dark uppercase text-sm tracking-wide">Weapons</span>
          </div>
          <div class="keys-row">
            <span class="key">1</span>
            <span class="key">2</span>
            <span class="key">3</span>
          </div>
          <p class="text-dark/50 text-xs mt-2">Switch between weapons</p>
        </div>

        <!-- Reload & Respawn -->
        <div class="instruction-block">
          <div class="flex items-center gap-2 mb-3">
            <div class="icon-box bg-gold">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-dark" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            </div>
            <span class="font-bold text-gold-dark uppercase text-sm tracking-wide">Reload & Respawn</span>
          </div>
          <div class="info-box rounded-lg p-3">
            <p class="text-dark/70 text-xs leading-relaxed">
              Answer <span class="text-teal-dark font-bold">3 questions</span> to reload or respawn!
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700;800&display=swap');

.home-page {
  font-family: 'Outfit', sans-serif;
}

/* Desert Sky Gradient */
.desert-sky {
  background: linear-gradient(
    180deg,
    #1a2f4a 0%,
    #2d4a6a 15%,
    #4a6a7a 30%,
    #7a8a6a 50%,
    #a89060 70%,
    #c4a060 85%,
    #8a6a40 100%
  );
}

/* Sun */
.sun {
  position: absolute;
  top: 12%;
  left: 50%;
  transform: translateX(-50%);
  width: 120px;
  height: 120px;
  background: radial-gradient(circle, #FACE68 0%, #FAAC68 50%, transparent 70%);
  border-radius: 50%;
}

.sun-glow {
  position: absolute;
  top: 8%;
  left: 50%;
  transform: translateX(-50%);
  width: 300px;
  height: 300px;
  background: radial-gradient(circle, rgba(250, 206, 104, 0.3) 0%, rgba(250, 172, 104, 0.1) 40%, transparent 70%);
  border-radius: 50%;
}

/* Sand Ground */
.sand-ground {
  background: linear-gradient(
    180deg,
    #8a6a40 0%,
    #9a7a50 30%,
    #a88a60 60%,
    #b89a70 100%
  );
}

/* Cactus styling */
.cactus {
  filter: drop-shadow(2px 4px 6px rgba(0, 0, 0, 0.3));
}

.cactus-small {
  filter: drop-shadow(1px 2px 3px rgba(0, 0, 0, 0.3));
}

.cactus-shadow {
  transform: translateY(0);
}

/* Tumbleweeds */
.tumbleweed {
  position: fixed;
  bottom: 100px;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: radial-gradient(circle at 30% 30%, #8a7050 0%, #6a5040 50%, #5a4030 100%);
  border: 3px solid #5a4030;
  opacity: 0.7;
  pointer-events: none;
  z-index: 5;
}

.tumbleweed::before {
  content: '';
  position: absolute;
  inset: 5px;
  border-radius: 50%;
  border: 2px dashed #4a3020;
}

.tumbleweed-1 {
  animation: tumble 25s linear infinite;
  animation-delay: 0s;
  width: 35px;
  height: 35px;
}

.tumbleweed-2 {
  animation: tumble 30s linear infinite;
  animation-delay: -10s;
  width: 28px;
  height: 28px;
  bottom: 90px;
}

.tumbleweed-3 {
  animation: tumble 20s linear infinite;
  animation-delay: -5s;
  width: 45px;
  height: 45px;
  bottom: 110px;
}

@keyframes tumble {
  0% {
    left: -60px;
    transform: rotate(0deg);
    opacity: 0;
  }
  5% {
    opacity: 0.7;
  }
  95% {
    opacity: 0.7;
  }
  100% {
    left: calc(100% + 60px);
    transform: rotate(720deg);
    opacity: 0;
  }
}

/* Dust particles */
.dust {
  position: absolute;
  border-radius: 50%;
  background: rgba(180, 150, 100, 0.4);
  pointer-events: none;
}

.dust-1 {
  width: 4px;
  height: 4px;
  bottom: 120px;
  animation: float-dust 8s ease-in-out infinite;
}

.dust-2 {
  width: 3px;
  height: 3px;
  bottom: 100px;
  animation: float-dust 10s ease-in-out infinite;
  animation-delay: -2s;
}

.dust-3 {
  width: 5px;
  height: 5px;
  bottom: 140px;
  animation: float-dust 12s ease-in-out infinite;
  animation-delay: -4s;
}

.dust-4 {
  width: 3px;
  height: 3px;
  bottom: 90px;
  animation: float-dust 9s ease-in-out infinite;
  animation-delay: -6s;
}

.dust-5 {
  width: 4px;
  height: 4px;
  bottom: 130px;
  animation: float-dust 11s ease-in-out infinite;
  animation-delay: -3s;
}

@keyframes float-dust {
  0% {
    left: -10px;
    opacity: 0;
    transform: translateY(0);
  }
  10% {
    opacity: 0.5;
  }
  50% {
    transform: translateY(-20px);
  }
  90% {
    opacity: 0.5;
  }
  100% {
    left: 100%;
    opacity: 0;
    transform: translateY(0);
  }
}

/* Color Variables */
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

.border-teal { border-color: #5A9CB5; }
.border-coral { border-color: #FA6868; }

/* Cards - light cream interior */
.card-main {
  background: linear-gradient(145deg, #f5f0e6 0%, #ebe5d9 100%);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(200, 180, 150, 0.5);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.4);
}

.card-secondary {
  background: linear-gradient(145deg, #f5f0e6 0%, #ebe5d9 100%);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(200, 180, 150, 0.5);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.3);
}

/* Logo */
.logo-icon {
  background: linear-gradient(135deg, #FACE68 0%, #FA6868 100%);
  border: 2px solid rgba(250, 206, 104, 0.5);
  box-shadow: 0 8px 25px rgba(250, 104, 104, 0.3);
}

/* Input fields */
.input-field {
  background: white;
  border: 2px solid #d4cfc5;
  color: #2c3e50;
  transition: all 0.3s ease;
}

.input-field:focus {
  border-color: #5A9CB5;
  box-shadow: 0 0 20px rgba(90, 156, 181, 0.2);
}

.input-field::placeholder {
  color: rgba(0, 0, 0, 0.4);
}

/* Buttons */
.btn-primary {
  background: linear-gradient(135deg, #FA6868 0%, #d94545 100%);
  color: white;
  border: 2px solid rgba(250, 104, 104, 0.5);
  box-shadow: 0 10px 30px rgba(250, 104, 104, 0.3);
  transition: all 0.3s ease;
}

.btn-primary:hover {
  transform: translateY(-2px);
  box-shadow: 0 15px 40px rgba(250, 104, 104, 0.4);
}

.btn-secondary {
  background: linear-gradient(135deg, #5A9CB5 0%, #4a8299 100%);
  color: white;
  border: 2px solid rgba(90, 156, 181, 0.5);
  box-shadow: 0 10px 30px rgba(90, 156, 181, 0.2);
  transition: all 0.3s ease;
}

.btn-secondary:hover {
  transform: translateY(-2px);
  box-shadow: 0 15px 40px rgba(90, 156, 181, 0.3);
}

/* Icon boxes */
.icon-box {
  width: 2rem;
  height: 2rem;
  border-radius: 0.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Info box */
.info-box {
  background: white;
  border: 1px solid #d4cfc5;
}

/* Instruction blocks */
.instruction-block {
  padding-bottom: 1rem;
  border-bottom: 1px solid rgba(200, 180, 150, 0.4);
}

.instruction-block:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

/* Keyboard key styling */
.key {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 2rem;
  height: 2rem;
  padding: 0 0.5rem;
  background: #5A9CB5;
  border: 1px solid #4a8a9f;
  border-radius: 0.375rem;
  font-size: 0.75rem;
  font-weight: 700;
  color: white;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.15);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.key-wide {
  min-width: auto;
  padding: 0 0.75rem;
}

.keys-row {
  display: flex;
  gap: 0.375rem;
  flex-wrap: wrap;
}

/* Title styling */
h1 {
  font-weight: 800;
  letter-spacing: 0.1em;
}

/* Responsive adjustments */
@media (max-width: 900px) {
  .home-page > div > .flex {
    flex-direction: column;
    align-items: center;
  }
  
  .cactus-shadow {
    display: none;
  }
}
</style>
