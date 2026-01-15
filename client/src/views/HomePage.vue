<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useWS } from '../composables/useWS';
import { useRouter } from 'vue-router';
import { usePlayerStore } from '../stores/playerStore';

const { init, send, on } = useWS();
const router = useRouter();
const playerStore = usePlayerStore();


const code = ref('');
const name = ref('');

const isValidCode = computed(() => {
  return !!code.value && !!name.value && code.value.length === 6;
});

function createRoom() {
  on('roomCreated', (message: any) => {
    router.push(`/dashboard/${message.roomCode}`);
  });
  send('createRoom');
}

function joinRoom() {
  on('joinedRoom', (message: any) => {
    playerStore.setPlayerData(message.player);
    router.push(`/game/${code.value}/${message.playerId}`)
  })
  send('joinRoom', { code: code.value, name: name.value })
}

onMounted(() => {
  init();
});
</script>

<template>
  <div class="w-screen h-screen bg-slate-900 flex justify-center items-center">
    <div class="bg-slate-800 p-12 rounded-xl text-white flex flex-col gap-8 min-w-120 items-center">
      <div class="font-bold text-2xl">WayWar</div>
      <input 
      class="border-2 border-slate-500 py-3 px-6 font-bold rounded-xl outline-none text-center focus:border-pink-500 w-full"
      v-model="code"
      name="code" 
      id="code"
      placeholder="Please enter your join code" 
      />
      <input 
        class="border-2 border-slate-500 py-3 px-6 font-bold rounded-xl outline-none text-center focus:border-pink-500 w-full"
        v-model="name"
        name="name" 
        id="name"
        placeholder="Please enter your name" 
      />
      <button 
        class="px-8 py-3 bg-pink-500 rounded-lg font-bold cursor-pointer w-full"
          :class="{
          'opacity-100': isValidCode,
          'opacity-50 pointer-events-none': !isValidCode
        }"
        @click="joinRoom"
      >
        JOIN GAME
      </button>
      <div class="w-full border-b-2 border-slate-500 border-dashed"></div>
      <button 
        class="px-8 py-3 bg-white rounded-lg font-bold text-slate-800 cursor-pointer w-full"
        @click="createRoom"
      >
        CREATE ROOM
      </button>
    </div>
  </div>
</template>