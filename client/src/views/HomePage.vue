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
  // Store terrain data in sessionStorage for the game page to use
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
  <div class="home-page w-screen h-screen bg-slate-950 flex justify-center items-center relative overflow-hidden">
    <!-- Ambient background glow -->
    <div class="fixed inset-0 pointer-events-none overflow-hidden">
      <div class="absolute -top-40 -left-40 w-96 h-96 bg-purple-600/20 rounded-full blur-3xl"></div>
      <div class="absolute -bottom-40 -right-40 w-96 h-96 bg-cyan-600/20 rounded-full blur-3xl"></div>
    </div>

    <div class="bg-slate-900/80 backdrop-blur-xl p-12 rounded-2xl text-white flex flex-col gap-8 min-w-120 items-center border border-slate-700/50 shadow-2xl relative z-10">
      <!-- Logo & Title -->
      <div class="flex items-center gap-4">
        <div class="w-14 h-14 rounded-xl bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center shadow-lg shadow-violet-500/25">
          <span class="text-white text-3xl font-bold">W</span>
        </div>
        <h1 class="text-4xl font-bold tracking-wide uppercase">
          <span class="bg-gradient-to-r from-cyan-400 to-blue-500 bg-clip-text text-transparent">WAY</span>
          <span class="bg-gradient-to-r from-fuchsia-400 to-pink-500 bg-clip-text text-transparent">WAR</span>
        </h1>
      </div>
      
      <!-- Error Message -->
      <div 
        v-if="error" 
        class="w-full bg-red-500/20 border border-red-500 text-red-400 px-4 py-3 rounded-lg text-center text-sm"
      >
        {{ error }}
      </div>

      <input 
        class="border-2 py-3 px-6 font-bold rounded-xl outline-none text-center focus:border-pink-500 w-full bg-slate-800/50"
        :class="error ? 'border-red-500' : 'border-slate-600'"
        v-model="code"
        name="code" 
        id="code"
        placeholder="Please enter your join code"
        @input="clearError"
      />
      <input 
        class="border-2 border-slate-600 py-3 px-6 font-bold rounded-xl outline-none text-center focus:border-pink-500 w-full bg-slate-800/50"
        v-model="name"
        name="name" 
        id="name"
        placeholder="Please enter your name"
        @input="clearError"
      />
      <button 
        class="px-8 py-3 bg-gradient-to-r from-fuchsia-500 to-pink-500 hover:from-fuchsia-400 hover:to-pink-400 rounded-xl font-bold cursor-pointer w-full transition-all shadow-lg shadow-fuchsia-500/30"
        :class="{
          'opacity-100': isValidCode && !isLoading,
          'opacity-50 pointer-events-none': !isValidCode || isLoading
        }"
        @click="joinRoom"
        :disabled="!isValidCode || isLoading"
      >
        {{ isLoading ? 'JOINING...' : 'JOIN GAME' }}
      </button>
      <div class="w-full border-b-2 border-slate-600 border-dashed"></div>
      <button 
        class="px-8 py-3 bg-gradient-to-r from-slate-200 to-white hover:from-white hover:to-slate-100 rounded-xl font-bold text-slate-800 cursor-pointer w-full transition-all shadow-lg"
        :class="{ 'opacity-50 pointer-events-none': isLoading }"
        @click="createRoom"
        :disabled="isLoading"
      >
        {{ isLoading ? 'CREATING...' : 'CREATE ROOM' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Rajdhani:wght@400;500;600;700&display=swap');

.home-page {
  font-family: 'Rajdhani', sans-serif;
}

input::placeholder {
  color: rgba(148, 163, 184, 0.7);
}
</style>