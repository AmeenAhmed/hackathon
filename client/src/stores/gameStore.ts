import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Player, GameState } from '../types'

export const useGameStore = defineStore('game', () => {
  const players = ref<Record<string, Player>>({})
  const gamePhase = ref<'waiting' | 'playing' | 'ended'>('waiting')
  const timer = ref(0)
  const score = ref<Record<string, number>>({})

  const playersList = computed(() => Object.values(players.value))
  const playerCount = computed(() => playersList.value.length)

  function setGameState(data: GameState) {
    players.value = data.players
    gamePhase.value = data.gamePhase
    timer.value = data.timer
    score.value = data.score
  }

  function updatePlayer(playerId: string, playerData: Partial<Player>) {
    if (players.value[playerId]) {
      players.value[playerId] = { ...players.value[playerId], ...playerData }
    }
  }

  function addPlayer(player: Player) {
    players.value[player.id] = player
  }

  function removePlayer(playerId: string) {
    delete players.value[playerId]
  }

  function getPlayer(playerId: string): Player | undefined {
    return players.value[playerId]
  }

  function reset() {
    players.value = {}
    gamePhase.value = 'waiting'
    timer.value = 0
    score.value = {}
  }

  return {
    // State
    players,
    gamePhase,
    timer,
    score,
    // Computed
    playersList,
    playerCount,
    // Actions
    setGameState,
    updatePlayer,
    addPlayer,
    removePlayer,
    getPlayer,
    reset,
  }
})
