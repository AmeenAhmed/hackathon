import Phaser from 'phaser';
import DashboardScene from './scenes/DashboardScene';
import type { GameState } from '../types';

export default class DashboardManager {
  private game: Phaser.Game | null = null;
  private config: Phaser.Types.Core.GameConfig;
  private ws: any;
  private pendingGameState: GameState | null = null;
  private sceneReady: boolean = false;

  constructor() {
    this.config = {
      type: Phaser.AUTO,
      parent: 'dashboard-container',
      backgroundColor: '#2d2d2d',
      pixelArt: true,
      antialias: false,
      render: {
        pixelArt: true,
        antialias: false,
        roundPixels: true
      },
      physics: {
        default: 'arcade',
        arcade: {
          gravity: { y: 0 },
          debug: false
        }
      },
      scale: {
        mode: Phaser.Scale.RESIZE,
        autoCenter: Phaser.Scale.CENTER_BOTH,
        width: window.innerWidth,
        height: window.innerHeight
      },
      scene: []
    };
  }

  init(ws: any): void {
    this.ws = ws;

    // Check if parent element exists
    const parentElement = document.getElementById('dashboard-container');
    if (!parentElement) {
      console.error('Parent element "dashboard-container" not found!');
      return;
    }
    console.log('Dashboard parent element found:', parentElement);

    // Add scene with data
    this.config.scene = [
      new DashboardScene({
        ws: this.ws
      })
    ];

    console.log('Creating Phaser game for dashboard with config:', this.config);

    // Create the game
    this.game = new Phaser.Game(this.config);

    // Listen for scene ready event
    this.game.events.once('ready', () => {
      console.log('Phaser game ready');
      // Wait for scene to be fully created
      const checkScene = () => {
        if (this.game && this.game.scene.isActive('DashboardScene')) {
          this.sceneReady = true;
          console.log('DashboardScene is now active');
          // Apply any pending game state
          if (this.pendingGameState) {
            this.handleGameUpdate(this.pendingGameState);
            this.pendingGameState = null;
          }
        } else {
          setTimeout(checkScene, 100);
        }
      };
      checkScene();
    });

    console.log('Phaser dashboard game created:', this.game);
  }

  destroy(): void {
    if (this.game) {
      this.game.destroy(true);
      this.game = null;
    }
    this.sceneReady = false;
    this.pendingGameState = null;
  }

  getGame(): Phaser.Game | null {
    return this.game;
  }

  private getScene(): DashboardScene | null {
    if (!this.game) return null;
    
    // Try to get the scene regardless of active state
    const scene = this.game.scene.getScene('DashboardScene') as DashboardScene;
    return scene || null;
  }

  handleGameUpdate(gameState: GameState): void {
    console.log('DashboardManager handleGameUpdate called', gameState);
    
    const scene = this.getScene();
    if (scene && this.sceneReady) {
      scene.updateGameState(gameState);
    } else {
      // Store pending state to apply when scene is ready
      console.log('Scene not ready, storing pending game state');
      this.pendingGameState = gameState;
    }
  }

  // Focus on a specific player by ID
  focusOnPlayer(playerId: string): void {
    const scene = this.getScene();
    if (scene && this.sceneReady) {
      scene.focusOnPlayer(playerId);
    }
  }

  // Handle bullet spawn
  handleBulletSpawn(data: any): void {
    const scene = this.getScene();
    if (scene && this.sceneReady) {
      scene.spawnBullet(data);
    }
  }

  // Handle bullet destroy
  handleBulletDestroy(data: any): void {
    const scene = this.getScene();
    if (scene && this.sceneReady) {
      scene.destroyBullet(data.bulletId);
    }
  }

  // Handle player hit
  handlePlayerHit(data: any): void {
    const scene = this.getScene();
    if (scene && this.sceneReady) {
      scene.handlePlayerHit(data);
    }
  }

  // Handle player death
  handlePlayerDeath(data: any): void {
    const scene = this.getScene();
    if (scene && this.sceneReady) {
      scene.handlePlayerDeath(data.playerId);
    }
  }

  // Handle player respawn
  handlePlayerRespawn(data: any): void {
    const scene = this.getScene();
    if (scene && this.sceneReady) {
      scene.handlePlayerRespawn(data.playerId, data.x, data.y);
    }
  }

  // Play countdown and start game
  playCountdownAndStart(onComplete: () => void): void {
    const scene = this.getScene();
    if (scene && this.sceneReady) {
      scene.playCountdownAndStart(onComplete);
    } else {
      // If scene not ready, just call the callback immediately
      onComplete();
    }
  }
}
