import Phaser from 'phaser';
import type { Player, GameState } from '../../types';

interface SceneData {
  ws: any;
}

interface PlayerSprite extends Phaser.Physics.Arcade.Sprite {
  playerId?: string;
  nameText?: Phaser.GameObjects.Text;
  gunSprite?: Phaser.GameObjects.Sprite;
  isDead?: boolean;
  deathText?: Phaser.GameObjects.Text;
}

export default class DashboardScene extends Phaser.Scene {
  private ws!: any;
  private players: Map<string, PlayerSprite>;
  private bullets: Map<string, Phaser.GameObjects.Sprite>;
  private tilemap!: Phaser.Tilemaps.Tilemap;
  private terrainLayer!: Phaser.Tilemaps.TilemapLayer;
  private objectsLayer!: Phaser.Tilemaps.TilemapLayer;
  private worldWidth: number = 3200;  // 200 tiles * 16 pixels
  private worldHeight: number = 3200; // 200 tiles * 16 pixels
  private currentZoom: number = 3;
  private zoomOutFactor: number = 2; // How much to zoom out (2x means half the zoom)
  private bgMusic: Phaser.Sound.BaseSound | null = null;

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
    this.bullets = new Map();
  }

  init(data: SceneData): void {
    this.ws = data.ws;
  }

  preload(): void {
    // Load terrain tileset - has 11 tiles (indices 0-10)
    this.load.spritesheet('terrain-tiles', '/assets/spritesheets/Hackathon-Terrain.png', {
      frameWidth: 16,
      frameHeight: 16,
      spacing: 0,
      margin: 0,
      startFrame: 0,
      endFrame: 10
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

    // Load bullets sprite (note: plural, not singular)
    this.load.spritesheet('bullets', '/assets/spritesheets/Hackathon-Bullet.png', {
      frameWidth: 16,
      frameHeight: 16
    });

    // Load countdown and game start sounds (dashboard only)
    this.load.audio('countdown', '/assets/sounds/3_2_1.wav');
    this.load.audio('gamestart', '/assets/sounds/deathmatch.wav');

    // Load background music (dashboard only)
    this.load.audio('bgmusic', '/assets/music/bg_music.wav');
  }

  create(): void {
    console.log('DashboardScene create() called');

    // Define tilemap dimensions - always 200x200 to match server
    const tileSize = 16;
    const mapWidth = 200;  // Fixed to match server map size
    const mapHeight = 200; // Fixed to match server map size

    // Set world bounds to match the full map
    this.physics.world.setBounds(0, 0, mapWidth * tileSize, mapHeight * tileSize);

    // Create the tilemap
    this.createTilemap(mapWidth, mapHeight, tileSize);

    // Create player animations
    this.createPlayerAnimations();

    // Setup camera - Use the actual map size (200 tiles * 16 pixels)
    this.cameras.main.setBounds(0, 0, mapWidth * tileSize, mapHeight * tileSize);
    this.cameras.main.setZoom(this.currentZoom);
    this.cameras.main.setRoundPixels(true);

    // Start at center of world
    this.cameras.main.centerOn((mapWidth * tileSize) / 2, (mapHeight * tileSize) / 2);

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

    // Note: WebSocket listeners are handled at DashboardPage level and forwarded here
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

    // Debug: Check how many tiles are in the tileset
    if (tileset) {
      console.log(`Dashboard tileset loaded with ${tileset.total} tiles (firstgid: ${tileset.firstgid})`);
    }

    // Create a layer for the terrain
    this.terrainLayer = this.tilemap.createBlankLayer('terrain', tileset!, 0, 0, mapWidth, mapHeight);

    // Get terrain and map objects data from server (passed via sessionStorage)
    const mapDataStr = sessionStorage.getItem('mapData');
    let terrainData: number[][] | null = null;
    let mapObjects: any[] | null = null;

    if (mapDataStr) {
      try {
        const mapData = JSON.parse(mapDataStr);
        if (mapData && mapData.terrain) {
          terrainData = mapData.terrain;
          // Get map objects (walls, cactus, chests, loot)
          if (mapData.mapObjects) {
            mapObjects = mapData.mapObjects;
            console.log(`Dashboard found ${mapObjects.length} map objects from server`);
          }
        }
      } catch (e) {
        console.error('Failed to parse terrain data:', e);
      }
    }

    // Use server terrain data or fallback to random generation
    if (terrainData && terrainData.length > 0) {
      // Use server-provided terrain
      const serverMapSize = terrainData.length;
      for (let y = 0; y < Math.min(mapHeight, serverMapSize); y++) {
        for (let x = 0; x < Math.min(mapWidth, serverMapSize); x++) {
          const tileIndex = terrainData[y][x] || 0;
          this.terrainLayer.putTileAt(tileIndex, x, y);
        }
      }
    } else {
      // Fallback: Generate random terrain
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
    }

    this.terrainLayer.setDepth(0);
    this.terrainLayer.setSize(mapWidth * tileSize, mapHeight * tileSize);

    // Create an objects layer for walls, cactus, chests, etc.
    this.objectsLayer = this.tilemap.createBlankLayer('objects', tileset!, 0, 0, mapWidth, mapHeight);
    this.objectsLayer.setDepth(1); // Above terrain but below players

    // Render map objects from server data
    if (mapObjects && mapObjects.length > 0) {
      console.log(`Dashboard rendering ${mapObjects.length} map objects`);
      for (const obj of mapObjects) {
        const objId = parseInt(obj.id);

        // Server sends: "7"=wall, "8"=wall2, "9"=cactus, "10"=chest (ignored), "11"=ammo, "12"=health
        // Only render walls and cacti (7-9), skip chests (10)
        if (objId >= 7 && objId <= 9) {
          // Use the actual tile indices from the spritesheet
          if (obj.x >= 0 && obj.x < mapWidth && obj.y >= 0 && obj.y < mapHeight) {
            this.objectsLayer.putTileAt(objId, obj.x, obj.y);
          }
        }
        // Skip chest rendering (objId === 10) and loot (11-12) for now
      }
    }

    // Set up collision for walls and cacti (tiles with ID 7, 8, 9) - for players only
    this.objectsLayer.setCollisionBetween(7, 9);
  }

  public updateGameState(gameState: GameState): void {
    if (!gameState || !gameState.players) return;

    // If game is already playing and music hasn't started, start it
    if (gameState.gamePhase === 'playing' && !this.bgMusic) {
      this.bgMusic = this.sound.add('bgmusic', {
        loop: true,
        volume: 0.3
      });
      this.bgMusic.play();
    }

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
        sprite.deathText?.destroy();
        sprite.destroy();
        this.players.delete(playerId);
      }
    }

    // Update spectator info
    this.updateSpectatorInfo();

    // Focus on first player if we haven't focused on anyone yet
    this.focusOnFirstPlayerIfNeeded();
  }

  updatePlayer(playerId: string, playerData: any): void {
    let sprite = this.players.get(playerId);

    // Check if player is dead
    const isDead = playerData.isDead || playerData.health <= 0;

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
      sprite.isDead = isDead;

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
      const gunSprite = this.add.sprite(playerData.x, playerData.y, 'guns', playerData.currentGun || 0);
      gunSprite.setScale(1);
      gunSprite.setDepth(15);
      gunSprite.setOrigin(0.5, 0.5);
      sprite.gunSprite = gunSprite;

      // Add death text (initially hidden)
      const deathText = this.add.text(playerData.x, playerData.y, 'DEAD', {
        fontSize: '12px',
        color: '#ff0000',
        backgroundColor: '#000000',
        padding: { x: 4, y: 2 }
      });
      deathText.setOrigin(0.5, 0.5);
      deathText.setDepth(20);
      deathText.setVisible(false);
      sprite.deathText = deathText;

      this.players.set(playerId, sprite);
    } else {
      // Update death state
      sprite.isDead = isDead;

      // Update existing player position with smooth interpolation
      this.tweens.add({
        targets: sprite,
        x: playerData.x,
        y: playerData.y,
        duration: 100,
        ease: 'Linear'
      });

      // Update animation based on player state
      if (!isDead) {
        const animKey = playerData.animation === 'running' ? 'player-run' : 'player-idle';
        if (!sprite.anims.currentAnim || sprite.anims.currentAnim.key !== animKey) {
          sprite.play(animKey);
        }
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

      // Update gun sprite
      if (sprite.gunSprite) {
        // Update gun frame/type
        sprite.gunSprite.setFrame(playerData.currentGun || 0);

        // Update gun position - gun should be at player position
        this.tweens.add({
          targets: sprite.gunSprite,
          x: playerData.x,
          y: playerData.y,
          duration: 100,
          ease: 'Linear'
        });

        // Update gun rotation
        if (playerData.gunRotation !== undefined) {
          sprite.gunSprite.setRotation(playerData.gunRotation);
        }

        // Update gun flip (use setFlipX, not setFlipY)
        if (playerData.gunFlipped !== undefined) {
          sprite.gunSprite.setFlipX(playerData.gunFlipped);
        }
      }

      // Update death text position
      if (sprite.deathText) {
        this.tweens.add({
          targets: sprite.deathText,
          x: playerData.x,
          y: playerData.y,
          duration: 100,
          ease: 'Linear'
        });
      }
    }

    // Handle death/alive visualization
    if (isDead) {
      sprite.setAlpha(0.3);
      sprite.stop();
      if (sprite.gunSprite) sprite.gunSprite.setVisible(false);
      if (sprite.deathText) sprite.deathText.setVisible(true);
    } else {
      sprite.setAlpha(1);
      if (sprite.gunSprite) sprite.gunSprite.setVisible(true);
      if (sprite.deathText) sprite.deathText.setVisible(false);
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

  public focusOnPlayer(playerId: string): void {
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

  public spawnBullet(data: any): void {
    const { bulletId, ownerId, x, y, angle, gunType } = data;
    console.log('DashboardScene spawnBullet called:', { bulletId, x, y, angle });

    // Create bullet sprite (no physics for now to avoid collision issues)
    const bullet = this.add.sprite(x, y, 'bullets', 0) as any;
    bullet.setScale(1);
    bullet.setRotation(angle);
    bullet.setDepth(9);

    // Store bullet
    this.bullets.set(bulletId, bullet);

    // Calculate end position using angle
    const distance = 1500;
    const bulletSpeed = 500; // pixels per second
    const duration = distance / bulletSpeed * 1000;
    const endX = x + Math.cos(angle) * distance;
    const endY = y + Math.sin(angle) * distance;

    // Create tween for bullet movement
    const tween = this.tweens.add({
      targets: bullet,
      x: endX,
      y: endY,
      duration: duration,
      ease: 'Linear',
      onUpdate: () => {
        // Check wall collision during movement
        if (this.objectsLayer && bullet && bullet.active !== false) {
          const tileX = Math.floor(bullet.x / 16);
          const tileY = Math.floor(bullet.y / 16);
          const tile = this.objectsLayer.getTileAt(tileX, tileY);

          if (tile && tile.collides) {
            // Bullet hit a wall
            tween.stop();
            this.destroyBullet(bulletId);
          }
        }
      },
      onComplete: () => {
        this.destroyBullet(bulletId);
      }
    });
  }

  public destroyBullet(bulletId: string): void {
    const bullet = this.bullets.get(bulletId);
    if (bullet) {
      bullet.destroy();
      this.bullets.delete(bulletId);
    }
  }

  public handlePlayerHit(data: any): void {
    const { targetPlayerId } = data;
    const sprite = this.players.get(targetPlayerId);

    if (sprite) {
      // Store original tint
      const originalTint = sprite.tintTopLeft;

      // Flash white effect
      sprite.setTintFill(0xffffff);

      // Add camera shake if this is the focused player
      if (this.focusedPlayerId === targetPlayerId) {
        this.cameras.main.shake(100, 0.003);
      }

      this.time.delayedCall(100, () => {
        if (sprite && sprite.active) {
          // Restore original tint
          if (originalTint !== 0xffffff) {
            sprite.setTint(originalTint);
          } else {
            sprite.clearTint();
          }
        }
      });
    }
  }

  public handlePlayerDeath(playerId: string): void {
    const sprite = this.players.get(playerId);
    if (sprite) {
      sprite.isDead = true;
      sprite.setAlpha(0.3);
      sprite.stop();
      if (sprite.gunSprite) sprite.gunSprite.setVisible(false);
      if (sprite.deathText) {
        sprite.deathText.setVisible(true);
        // Pulse effect on death text
        this.tweens.add({
          targets: sprite.deathText,
          scaleX: 1.2,
          scaleY: 1.2,
          duration: 200,
          yoyo: true,
          repeat: 2
        });
      }
    }
  }

  public handlePlayerRespawn(playerId: string, x: number, y: number): void {
    const sprite = this.players.get(playerId);
    if (sprite) {
      sprite.isDead = false;
      sprite.setAlpha(1);
      sprite.setPosition(x, y);
      if (sprite.gunSprite) {
        sprite.gunSprite.setVisible(true);
        sprite.gunSprite.setPosition(x, y);
      }
      if (sprite.deathText) {
        sprite.deathText.setVisible(false);
        sprite.deathText.setPosition(x, y);
      }
      if (sprite.nameText) {
        sprite.nameText.setPosition(x, y - 20);
      }

      // Respawn effect
      sprite.setScale(0);
      this.tweens.add({
        targets: sprite,
        scaleX: 1,
        scaleY: 1,
        duration: 300,
        ease: 'Back.easeOut'
      });
    }
  }

  public playCountdownAndStart(onComplete: () => void): void {
    // Play countdown sound
    const countdownSound = this.sound.add('countdown');
    countdownSound.play();

    // Show countdown text
    const countdownText = this.add.text(
      this.cameras.main.width / 2,
      this.cameras.main.height / 2,
      'Starting in 3...',
      {
        fontSize: '48px',
        color: '#ffffff',
        backgroundColor: '#000000',
        padding: { x: 20, y: 10 }
      }
    );
    countdownText.setOrigin(0.5, 0.5);
    countdownText.setScrollFactor(0);
    countdownText.setDepth(200);

    // Countdown animation
    let count = 3;
    const countdownTimer = this.time.addEvent({
      delay: 1000,
      callback: () => {
        count--;
        if (count > 0) {
          countdownText.setText(`Starting in ${count}...`);
        } else if (count === 0) {
          countdownText.setText('GO!');
          // Play game start sound
          const gamestartSound = this.sound.add('gamestart');
          gamestartSound.play();

          // Start background music on loop
          if (!this.bgMusic) {
            this.bgMusic = this.sound.add('bgmusic', {
              loop: true,
              volume: 0.3  // Set to 30% volume so it's not too loud
            });
            this.bgMusic.play();
          }

          // Remove countdown text after a moment
          this.time.delayedCall(500, () => {
            countdownText.destroy();
            // Call the callback to actually start the game
            onComplete();
          });
        }
      },
      repeat: 3
    });
  }

  update(): void {
    // Update all player-related sprites positions
    for (const [playerId, sprite] of this.players) {
      if (sprite && sprite.active) {
        // Update name text position to follow sprite
        if (sprite.nameText) {
          sprite.nameText.setPosition(sprite.x, sprite.y - 20);
        }

        // Update gun sprite position to follow sprite (at player position)
        if (sprite.gunSprite && sprite.gunSprite.visible) {
          sprite.gunSprite.setPosition(sprite.x, sprite.y);
        }

        // Update death text position to follow sprite
        if (sprite.deathText && sprite.deathText.visible) {
          sprite.deathText.setPosition(sprite.x, sprite.y);
        }
      }
    }
  }

  destroy(): void {
    if (this.cameraRotationTimer) {
      this.cameraRotationTimer.destroy();
    }

    // Stop background music
    if (this.bgMusic) {
      this.bgMusic.stop();
      this.bgMusic.destroy();
      this.bgMusic = null;
    }

    // Clean up players
    for (const [playerId, sprite] of this.players) {
      if (sprite.nameText) sprite.nameText.destroy();
      if (sprite.gunSprite) sprite.gunSprite.destroy();
      if (sprite.deathText) sprite.deathText.destroy();
      sprite.destroy();
    }
    this.players.clear();

    // Clean up bullets
    for (const [bulletId, bullet] of this.bullets) {
      bullet.destroy();
    }
    this.bullets.clear();
  }
}
