import Phaser from 'phaser';
import MainScene from './scenes/MainScene';
import PreloadScene from './scenes/PreloadScene';
import UIScene from './scenes/UIScene';

export default class GameManager {
  private game: Phaser.Game | null = null;
  private config: Phaser.Types.Core.GameConfig;
  private roomCode: string;
  private playerId: string;
  private ws: any; // WebSocket connection from useWS composable

  constructor() {
    this.roomCode = '';
    this.playerId = '';
    this.config = {
      type: Phaser.WEBGL, // Force WebGL for better performance
      parent: 'game-container',
      backgroundColor: '#2d2d2d',
      pixelArt: true,  // Enable pixel art mode
      antialias: false, // Disable anti-aliasing for crisp pixels
      fps: {
        target: 60, // 60 FPS to match server tick rate
        forceSetTimeOut: true
      },
      render: {
        pixelArt: true,
        antialias: false,
        roundPixels: false // Disabled for performance
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

  init(roomCode: string, playerId: string, ws: any): void {
    this.roomCode = roomCode;
    this.playerId = playerId;
    this.ws = ws;

    // Check if parent element exists
    const parentElement = document.getElementById('game-container');
    if (!parentElement) {
      // console.error('Parent element "game-container" not found!');
      return;
    }
    // console.log('Parent element found:', parentElement);

    // Add scenes with data
    this.config.scene = [
      new PreloadScene({
        roomCode: this.roomCode,
        playerId: this.playerId,
        ws: this.ws
      }),
      new MainScene({
        roomCode: this.roomCode,
        playerId: this.playerId,
        ws: this.ws
      }),
      new UIScene()
    ];

    // console.log('Creating Phaser game with config:', this.config);

    // Create the game
    this.game = new Phaser.Game(this.config);

    // console.log('Phaser game created:', this.game);
  }

  destroy(): void {
    if (this.game) {
      this.game.destroy(true);
      this.game = null;
    }
  }

  getGame(): Phaser.Game | null {
    return this.game;
  }

  updatePlayerPosition(x: number, y: number, animation: string): void {
    if (this.game && this.game.scene.isActive('MainScene')) {
      const mainScene = this.game.scene.getScene('MainScene') as MainScene;
      mainScene.updateLocalPlayer(x, y, animation);
    }
  }

  handleGameUpdate(gameState: any): void {
    if (this.game && this.game.scene.isActive('MainScene')) {
      const mainScene = this.game.scene.getScene('MainScene') as MainScene;
      mainScene.updateGameState(gameState);
    }
  }
}