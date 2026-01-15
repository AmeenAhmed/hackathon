import Phaser from 'phaser';
import type { Player, GameState } from '../../types';

interface SceneData {
  ws: any;
}

interface PlayerSprite extends Phaser.Physics.Arcade.Sprite {
  playerId?: string;
  nameText?: Phaser.GameObjects.Text;
  gunSprite?: Phaser.GameObjects.Sprite;
}

export default class DashboardScene extends Phaser.Scene {
  private ws!: any;
  private players: Map<string, PlayerSprite>;
  private tilemap!: Phaser.Tilemaps.Tilemap;
  private terrainLayer!: Phaser.Tilemaps.TilemapLayer;
  private worldWidth: number = 3000;
  private worldHeight: number = 3000;
  private currentZoom: number = 3;
  private zoomOutFactor: number = 2; // How much to zoom out (2x means half the zoom)
  
  // Spectator camera rotation
  private playerIds: string[] = [];
  private currentFocusIndex: number = 0;
  private cameraRotationTimer: Phaser.Time.TimerEvent | null = null;
  private focusedPlayerId: string | null = null;
  private spectatorInfoText!: Phaser.GameObjects.Text;
  private isTransitioning: boolean = false;

  constructor(data?: SceneData) {
    super({ key: 'DashboardScene' });
    if (data) {
      this.ws = data.ws;
    }
    this.players = new Map();
  }

  init(data: SceneData): void {
    this.ws = data.ws;
  }

  preload(): void {
    // Load terrain tileset
    this.load.spritesheet('terrain-tiles', '/assets/spritesheets/Hackathon-Terrain.png', {
      frameWidth: 16,
      frameHeight: 16,
      spacing: 0,
      margin: 0
    });

    // Load player spritesheets
    this.load.spritesheet('player-idle', '/assets/spritesheets/Hackathon-Idle.png', {
      frameWidth: 16,
      frameHeight: 16
    });

    this.load.spritesheet('player-run', '/assets/spritesheets/Hackathon-Run.png', {
      frameWidth: 16,
      frameHeight: 16
    });

    // Load guns spritesheet
    this.load.spritesheet('guns', '/assets/spritesheets/Hackathon-Guns.png', {
      frameWidth: 16,
      frameHeight: 16
    });
  }

  create(): void {
    console.log('DashboardScene create() called');

    // Define tilemap dimensions
    const tileSize = 16;
    const mapWidth = Math.ceil(this.worldWidth / tileSize);
    const mapHeight = Math.ceil(this.worldHeight / tileSize);

    // Set world bounds
    this.physics.world.setBounds(0, 0, this.worldWidth, this.worldHeight);

    // Create the tilemap
    this.createTilemap(mapWidth, mapHeight, tileSize);

    // Create player animations
    this.createPlayerAnimations();

    // Setup camera
    this.cameras.main.setBounds(0, 0, this.worldWidth, this.worldHeight);
    this.cameras.main.setZoom(this.currentZoom);
    this.cameras.main.setRoundPixels(true);
    
    // Start at center of world
    this.cameras.main.centerOn(this.worldWidth / 2, this.worldHeight / 2);

    // Add spectator info text
    this.spectatorInfoText = this.add.text(10, 10, 'Spectator Mode - Waiting for players...', {
      fontSize: '16px',
      color: '#ffffff',
      backgroundColor: '#000000',
      padding: { x: 10, y: 5 }
    });
    this.spectatorInfoText.setScrollFactor(0).setDepth(100);

    // Handle window resize
    this.scale.on('resize', (gameSize: any) => {
      this.cameras.main.setSize(gameSize.width, gameSize.height);
    });

    // Start camera rotation timer (5 seconds)
    this.startCameraRotation();
  }

  createPlayerAnimations(): void {
    // Create idle animation
    if (!this.anims.exists('player-idle')) {
      this.anims.create({
        key: 'player-idle',
        frames: this.anims.generateFrameNumbers('player-idle', { start: 0, end: -1 }),
        frameRate: 8,
        repeat: -1
      });
    }

    // Create run animation
    if (!this.anims.exists('player-run')) {
      this.anims.create({
        key: 'player-run',
        frames: this.anims.generateFrameNumbers('player-run', { start: 0, end: -1 }),
        frameRate: 12,
        repeat: -1
      });
    }
  }

  createTilemap(mapWidth: number, mapHeight: number, tileSize: number): void {
    // Create a blank tilemap
    this.tilemap = this.make.tilemap({
      tileWidth: tileSize,
      tileHeight: tileSize,
      width: mapWidth,
      height: mapHeight
    });

    // Add the tileset to the map
    const tileset = this.tilemap.addTilesetImage('terrain', 'terrain-tiles', tileSize, tileSize, 0, 0);

    // Create a layer for the terrain
    this.terrainLayer = this.tilemap.createBlankLayer('terrain', tileset!, 0, 0, mapWidth, mapHeight);

    // Generate random terrain
    for (let y = 0; y < mapHeight; y++) {
      for (let x = 0; x < mapWidth; x++) {
        let tileIndex: number;
        const random = Math.random();

        if (random < 0.95) {
          tileIndex = 0;
        } else {
          tileIndex = Phaser.Math.Between(1, 6);
        }

        this.terrainLayer.putTileAt(tileIndex, x, y);
      }
    }

    this.terrainLayer.setDepth(0);
    this.terrainLayer.setSize(mapWidth * tileSize, mapHeight * tileSize);
  }

  updateGameState(gameState: GameState): void {
    console.log('DashboardScene updateGameState:', gameState);
    if (!gameState || !gameState.players) return;

    // Update player IDs list for camera rotation
    this.playerIds = Object.keys(gameState.players);

    // Update all players
    for (const [playerId, playerData] of Object.entries(gameState.players)) {
      this.updatePlayer(playerId, playerData);
    }

    // Remove players that are no longer in the game
    for (const [playerId, sprite] of this.players) {
      if (!gameState.players[playerId]) {
        sprite.nameText?.destroy();
        sprite.gunSprite?.destroy();
        sprite.destroy();
        this.players.delete(playerId);
      }
    }

    // Update spectator info
    this.updateSpectatorInfo();

    // Focus on first player if we haven't focused on anyone yet
    this.focusOnFirstPlayerIfNeeded();
  }

  updatePlayer(playerId: string, playerData: Player): void {
    let sprite = this.players.get(playerId);

    if (!sprite) {
      // Create new player sprite
      sprite = this.physics.add.sprite(
        playerData.x,
        playerData.y,
        'player-idle'
      ) as PlayerSprite;
      sprite.setScale(1);
      sprite.playerId = playerId;
      sprite.setDepth(10);

      // Start with idle animation
      sprite.play('player-idle');

      // Set initial direction
      if (playerData.direction === 'left') {
        sprite.setFlipX(true);
      } else {
        sprite.setFlipX(false);
      }

      // Set player color
      if (playerData.color) {
        sprite.setTint(parseInt(playerData.color.replace('#', '0x')));
      }

      // Add player name
      const nameText = this.add.text(
        playerData.x,
        playerData.y - 20,
        playerData.name || 'Player', {
        fontSize: '10px',
        color: '#ffffff',
        backgroundColor: '#000000',
        padding: { x: 2, y: 1 }
      });
      nameText.setOrigin(0.5, 0.5);
      nameText.setDepth(11);
      sprite.nameText = nameText;

      // Add gun sprite
      const gunSprite = this.add.sprite(playerData.x, playerData.y, 'guns', 0);
      gunSprite.setScale(1);
      gunSprite.setDepth(15);
      gunSprite.setOrigin(0.5, 0.5);
      sprite.gunSprite = gunSprite;

      this.players.set(playerId, sprite);
    } else {
      // Update existing player position with smooth interpolation
      this.tweens.add({
        targets: sprite,
        x: playerData.x,
        y: playerData.y,
        duration: 100,
        ease: 'Linear'
      });

      // Update animation based on player state
      const animKey = playerData.animation === 'running' ? 'player-run' : 'player-idle';
      if (!sprite.anims.currentAnim || sprite.anims.currentAnim.key !== animKey) {
        sprite.play(animKey);
      }

      // Update direction/flip
      if (playerData.direction === 'left') {
        sprite.setFlipX(true);
      } else if (playerData.direction === 'right') {
        sprite.setFlipX(false);
      }

      // Update name position
      if (sprite.nameText) {
        this.tweens.add({
          targets: sprite.nameText,
          x: playerData.x,
          y: playerData.y - 20,
          duration: 100,
          ease: 'Linear'
        });
      }

      // Update gun position
      if (sprite.gunSprite) {
        this.tweens.add({
          targets: sprite.gunSprite,
          x: playerData.x,
          y: playerData.y,
          duration: 100,
          ease: 'Linear'
        });
      }
    }
  }

  startCameraRotation(): void {
    // Clear existing timer if any
    if (this.cameraRotationTimer) {
      this.cameraRotationTimer.destroy();
    }

    // Create timer to switch focus every 6 seconds (accounting for ~1.8s transition time)
    this.cameraRotationTimer = this.time.addEvent({
      delay: 6000,
      callback: this.rotateCamera,
      callbackScope: this,
      loop: true
    });
  }

  rotateCamera(): void {
    if (this.playerIds.length === 0) {
      this.focusedPlayerId = null;
      return;
    }

    // Move to next player
    this.currentFocusIndex = (this.currentFocusIndex + 1) % this.playerIds.length;
    this.focusOnPlayer(this.playerIds[this.currentFocusIndex]);
  }

  // Focus on first available player (called when players join)
  private focusOnFirstPlayerIfNeeded(): void {
    if (!this.focusedPlayerId && this.playerIds.length > 0 && !this.isTransitioning) {
      this.currentFocusIndex = 0;
      this.focusOnPlayer(this.playerIds[0]);
    }
  }

  focusOnPlayer(playerId: string): void {
    const sprite = this.players.get(playerId);
    if (!sprite) return;

    // Prevent overlapping transitions
    if (this.isTransitioning) return;
    this.isTransitioning = true;

    this.focusedPlayerId = playerId;
    this.updateSpectatorInfo();

    // Stop following current player
    this.cameras.main.stopFollow();

    // Animation timing
    const zoomOutDuration = 500;
    const panDuration = 800;
    const zoomInDuration = 600;

    // Calculate zoom out level (2x zoom out from current)
    const zoomedOutLevel = this.currentZoom / this.zoomOutFactor;

    // Get current camera center position
    const startX = this.cameras.main.scrollX + this.cameras.main.width / 2;
    const startY = this.cameras.main.scrollY + this.cameras.main.height / 2;

    // Step 1: Zoom out 2x
    this.tweens.add({
      targets: this.cameras.main,
      zoom: zoomedOutLevel,
      duration: zoomOutDuration,
      ease: 'Sine.easeOut',
      onUpdate: () => {
        // Keep centered on current position during zoom out
        this.cameras.main.centerOn(startX, startY);
      },
      onComplete: () => {
        // Step 2: Pan to the target player using scroll tweens
        const targetScrollX = sprite.x - this.cameras.main.width / 2;
        const targetScrollY = sprite.y - this.cameras.main.height / 2;

        this.tweens.add({
          targets: this.cameras.main,
          scrollX: targetScrollX,
          scrollY: targetScrollY,
          duration: panDuration,
          ease: 'Sine.easeInOut',
          onComplete: () => {
            // Step 3: Zoom back in while manually centering on sprite
            this.tweens.add({
              targets: this.cameras.main,
              zoom: this.currentZoom,
              duration: zoomInDuration,
              ease: 'Sine.easeInOut',
              onUpdate: () => {
                // Keep camera centered on sprite during zoom
                this.cameras.main.centerOn(sprite.x, sprite.y);
              },
              onComplete: () => {
                // Step 4: Now start following smoothly
                this.cameras.main.startFollow(sprite, true, 0.05, 0.05);
                this.isTransitioning = false;
              }
            });
          }
        });
      }
    });
  }

  updateSpectatorInfo(): void {
    const playerCount = this.playerIds.length;
    const focusedPlayer = this.focusedPlayerId ? this.players.get(this.focusedPlayerId) : null;
    const focusedName = focusedPlayer?.nameText?.text || 'None';
    
    this.spectatorInfoText.setText(
      `Spectator Mode | Players: ${playerCount} | Watching: ${focusedName}`
    );
  }

  update(): void {
    // Follow the focused player if we have one
    if (this.focusedPlayerId) {
      const sprite = this.players.get(this.focusedPlayerId);
      if (sprite && sprite.nameText) {
        // Update name text position to follow sprite
        sprite.nameText.setPosition(sprite.x, sprite.y - 20);
      }
      if (sprite && sprite.gunSprite) {
        sprite.gunSprite.setPosition(sprite.x, sprite.y);
      }
    }
  }

  destroy(): void {
    if (this.cameraRotationTimer) {
      this.cameraRotationTimer.destroy();
    }
    this.players.clear();
  }
}
