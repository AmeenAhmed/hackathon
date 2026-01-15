import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useWS } from '../composables/useWS';

export const usePlayerStore = defineStore('player', () => {
  const ws = useWS();
  const name = ref('');
  const id = ref('');
  const color = ref('');
  const x = ref(0);
  const y = ref(0);
  const animation = ref<'idle' | 'running'>('idle');
  const isProtected = ref<boolean>(false);

  function setPlayerData(data: any) {
    name.value = data.name;
    id.value = data.id;
    color.value = data.color;
    x.value = data.x;
    y.value = data.y;
    animation.value = data.animation;
    isProtected.value = data.isProtected;
  }

  return {
    id,
    setPlayerData,
  }
});