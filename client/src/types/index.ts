export interface Player {
  id: string
  name: string
  color: string
  x: number
  y: number
  animation: 'idle' | 'running'
  direction: 'left' | 'right'
  isProtected: boolean
}

export interface GameState {
  players: Record<string, Player>
  gamePhase: 'waiting' | 'playing' | 'ended'
  timer: number
  score: Record<string, number>
}
