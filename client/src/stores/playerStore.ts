import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useWS } from '../composables/useWS';

export const useUserStore = defineStore('player', () => {
  const ws = useWS();
  const name = ref('');
  const id = ref('');
  const color = ref('');
  const x = ref('')
  const y = ref('')
});