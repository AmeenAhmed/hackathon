import Phaser from 'phaser';

interface SceneData {
  roomCode: string;
  playerId: string;
  ws: any;
}

interface Player {
  id: string;
  name: string;
  color: string;
  x: number;
  y: number;
  animation: string;
  isProtected: boolean;
  direction?: string;
}

interface PlayerSprite extends Phaser.Physics.Arcade.Sprite {
  playerId?: string;
  nameText?: Phaser.GameObjects.Text;
  gunSprite?: Phaser.GameObjects.Sprite;
}

export default class MainScene extends Phaser.Scene {
  private roomCode!: string;
  private playerId!: string;
  private ws!: any;
  private localPlayer!: PlayerSprite;
  private otherPlayers: Map<string, PlayerSprite>;
  private cursors!: Phaser.Types.Input.Keyboard.CursorKeys;
  private wasd!: {
    W: Phaser.Input.Keyboard.Key;
    A: Phaser.Input.Keyboard.Key;
    S: Phaser.Input.Keyboard.Key;
    D: Phaser.Input.Keyboard.Key;
  };
  private playerSpeed: number = 150;
  private lastPositionUpdate: { x: number; y: number; animation: string; direction: string };
  private updateThrottle: number = 50; // milliseconds
  private lastUpdateTime: number = 0;
  private tilemap!: Phaser.Tilemaps.Tilemap;
  private terrainLayer!: Phaser.Tilemaps.TilemapLayer;
  private worldWidth: number = 3000;
  private worldHeight: number = 3000;
  private currentZoom: number = 3;
  private zoomKeys!: {
    plus: Phaser.Input.Keyboard.Key;
    minus: Phaser.Input.Keyboard.Key;
  };
  private mouseWorldX: number = 0;
  private mouseWorldY: number = 0;

  constructor(data: SceneData) {
    super({ key: 'MainScene' });
    if (data) {
      this.roomCode = data.roomCode;
      this.playerId = data.playerId;
      this.ws = data.ws;
      console.log('MainScene created with data:', { roomCode: this.roomCode, playerId: this.playerId });
    }
    this.otherPlayers = new Map();
    this.lastPositionUpdate = { x: 0, y: 0, animation: 'idle', direction: 'right' };
  }

  init(data: SceneData): void {
    this.roomCode = data.roomCode;
    this.playerId = data.playerId;
    this.ws = data.ws;
  }

  create(): void {
    console.log('MainScene create() called');

    // Define tilemap dimensions
    const tileSize = 16;
    const mapWidth = Math.ceil(this.worldWidth / tileSize);
    const mapHeight = Math.ceil(this.worldHeight / tileSize);

    console.log(`Creating tilemap for ${this.worldWidth}x${this.worldHeight} world (${mapWidth}x${mapHeight} tiles)`);

    // Set world bounds larger than viewport
    this.physics.world.setBounds(0, 0, this.worldWidth, this.worldHeight);

    // Create the tilemap
    this.createTilemap(mapWidth, mapHeight, tileSize);

    // Create player animations
    this.createPlayerAnimations();

    // Create local player
    this.createLocalPlayer();

    // Setup camera to follow player in the larger world
    this.cameras.main.setBounds(0, 0, this.worldWidth, this.worldHeight);
    this.cameras.main.startFollow(this.localPlayer, true, 0.05, 0.05); // Smoother camera follow
    this.cameras.main.setZoom(this.currentZoom); // Zoom in closer to the player
    this.cameras.main.setRoundPixels(true); // Round camera position to avoid subpixel rendering

    // Setup input
    this.cursors = this.input.keyboard!.createCursorKeys();
    this.wasd = {
      W: this.input.keyboard!.addKey('W'),
      A: this.input.keyboard!.addKey('A'),
      S: this.input.keyboard!.addKey('S'),
      D: this.input.keyboard!.addKey('D')
    };

    // Setup zoom controls (+ and - keys)
    this.zoomKeys = {
      plus: this.input.keyboard!.addKey(Phaser.Input.Keyboard.KeyCodes.PLUS),
      minus: this.input.keyboard!.addKey(Phaser.Input.Keyboard.KeyCodes.MINUS)
    };

    // Also support numpad + and -
    const numpadPlus = this.input.keyboard!.addKey(Phaser.Input.Keyboard.KeyCodes.NUMPAD_ADD);
    const numpadMinus = this.input.keyboard!.addKey(Phaser.Input.Keyboard.KeyCodes.NUMPAD_SUBTRACT);

    // Handle zoom controls (use integer zoom levels for pixel art)
    this.zoomKeys.plus.on('down', () => this.adjustZoom(1));
    this.zoomKeys.minus.on('down', () => this.adjustZoom(-1));
    numpadPlus.on('down', () => this.adjustZoom(1));
    numpadMinus.on('down', () => this.adjustZoom(-1));

    // Display room code
    this.add.text(10, 10, `Room: ${this.roomCode}`, {
      fontSize: '20px',
      color: '#ffffff',
      backgroundColor: '#000000',
      padding: { x: 10, y: 5 }
    }).setScrollFactor(0).setDepth(100);

    // Display player ID
    this.add.text(10, 40, `Player ID: ${this.playerId}`, {
      fontSize: '16px',
      color: '#ffffff',
      backgroundColor: '#000000',
      padding: { x: 10, y: 5 }
    }).setScrollFactor(0).setDepth(100);

    // Display controls
    this.add.text(10, 70, 'Move: WASD/Arrows | Zoom: +/- | Aim: Mouse', {
      fontSize: '14px',
      color: '#ffffff',
      backgroundColor: '#000000',
      padding: { x: 10, y: 5 }
    }).setScrollFactor(0).setDepth(100);

    // Add crosshair cursor
    this.input.setDefaultCursor('crosshair');

    // Listen for game updates from WebSocket
    this.setupWebSocketListeners();

    // Handle window resize
    this.scale.on('resize', (gameSize: any) => {
      this.cameras.main.setSize(gameSize.width, gameSize.height);
    });
  }

  createPlayerAnimations(): void {
    // Create idle animation
    this.anims.create({
      key: 'player-idle',
      frames: this.anims.generateFrameNumbers('player-idle', { start: 0, end: -1 }),
      frameRate: 8,
      repeat: -1
    });

    // Create run animation
    this.anims.create({
      key: 'player-run',
      frames: this.anims.generateFrameNumbers('player-run', { start: 0, end: -1 }),
      frameRate: 12,
      repeat: -1
    });
  }

  createTilemap(mapWidth: number, mapHeight: number, tileSize: number): void {
    // Create a blank tilemap
    this.tilemap = this.make.tilemap({
      tileWidth: tileSize,
      tileHeight: tileSize,
      width: mapWidth,
      height: mapHeight
    });

    // Add the tileset to the map (no margin or spacing between tiles)
    const tileset = this.tilemap.addTilesetImage('terrain', 'terrain-tiles', tileSize, tileSize, 0, 0);

    // Create a layer for the terrain at position 0,0
    this.terrainLayer = this.tilemap.createBlankLayer('terrain', tileset!, 0, 0, mapWidth, mapHeight);

    // Generate random terrain with heavy bias towards tile 0
    for (let y = 0; y < mapHeight; y++) {
      for (let x = 0; x < mapWidth; x++) {
        // Tile 0 appears 20x more often than all other tiles combined
        // Ratio 20:1 means tile 0 = 95%, others = 5% total
        let tileIndex: number;
        const random = Math.random();

        if (random < 0.95) {
          // 95% chance for tile 0 (base/ground tile)
          tileIndex = 0;
        } else {
          // 5% chance split among tiles 1-6 (sparse decoration tiles)
          tileIndex = Phaser.Math.Between(1, 6);
        }

        this.terrainLayer.putTileAt(tileIndex, x, y);
      }
    }

    // Set the terrain layer to be below everything else
    this.terrainLayer.setDepth(0);

    // Make sure the layer is sized correctly
    this.terrainLayer.setSize(mapWidth * tileSize, mapHeight * tileSize);

    // Ensure pixel-perfect rendering for the tilemap
    this.terrainLayer.setRenderOrder();

    console.log(`Tilemap created: ${mapWidth}x${mapHeight} tiles (${tileSize}px each) = ${mapWidth * tileSize}x${mapHeight * tileSize}px total`);
  }

  createLocalPlayer(): void {
    // Create the player sprite at the center of the world
    const startX = this.worldWidth / 2;
    const startY = this.worldHeight / 2;
    this.localPlayer = this.physics.add.sprite(startX, startY, 'player-idle') as PlayerSprite;
    this.localPlayer.setCollideWorldBounds(true);
    this.localPlayer.setScale(1); // Keep original 16x16 size
    this.localPlayer.playerId = this.playerId;
    this.localPlayer.setDepth(10); // Above terrain

    // Start with idle animation
    this.localPlayer.play('player-idle');

    // Set player color if available from store
    const playerStore = (window as any).playerStore;
    if (playerStore && playerStore.color) {
      this.localPlayer.setTint(parseInt(playerStore.color.replace('#', '0x')));
    }

    // Add player name text
    const nameText = this.add.text(0, -20, 'You', {
      fontSize: '10px',
      color: '#ffffff',
      backgroundColor: '#000000',
      padding: { x: 2, y: 1 }
    });
    nameText.setOrigin(0.5, 0.5);
    nameText.setDepth(11); // Above player
    this.localPlayer.nameText = nameText;

    // Add gun sprite (use frame 0, first gun)
    const gunSprite = this.add.sprite(startX, startY, 'guns', 0);
    gunSprite.setScale(1);
    gunSprite.setDepth(15); // Higher depth to ensure it's always on top
    gunSprite.setOrigin(0.5, 0.5);
    this.localPlayer.gunSprite = gunSprite;
  }

  setupWebSocketListeners(): void {
    // Listen for game state updates
    if (this.ws) {
      this.ws.on('gameUpdate', (data: any) => {
        this.updateGameState(data.gameState);
      });

      this.ws.on('initialState', (data: any) => {
        this.updateGameState(data.gameState);
      });
    }
  }

  updateGameState(gameState: any): void {
    if (!gameState || !gameState.players) return;

    // Update other players
    for (const [playerId, playerData] of Object.entries(gameState.players)) {
      if (playerId !== this.playerId) {
        this.updateOtherPlayer(playerId, playerData as Player);
      }
    }

    // Remove players that are no longer in the game
    for (const [playerId, sprite] of this.otherPlayers) {
      if (!gameState.players[playerId]) {
        sprite.nameText?.destroy();
        sprite.gunSprite?.destroy();
        sprite.destroy();
        this.otherPlayers.delete(playerId);
      }
    }
  }

  updateOtherPlayer(playerId: string, playerData: Player): void {
    let sprite = this.otherPlayers.get(playerId);

    if (!sprite) {
      // Create new player sprite
      sprite = this.physics.add.sprite(
        playerData.x,
        playerData.y,
        'player-idle'
      ) as PlayerSprite;
      sprite.setScale(1);
      sprite.playerId = playerId;
      sprite.setDepth(10); // Above terrain

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
      nameText.setDepth(11); // Above player
      sprite.nameText = nameText;

      // Add gun sprite for other players
      const gunSprite = this.add.sprite(playerData.x, playerData.y, 'guns', 0);
      gunSprite.setScale(1);
      gunSprite.setDepth(15); // Higher depth to ensure it's always on top
      gunSprite.setOrigin(0.5, 0.5);
      sprite.gunSprite = gunSprite;

      this.otherPlayers.set(playerId, sprite);
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

  updateLocalPlayer(x: number, y: number, animation: string): void {
    if (this.localPlayer) {
      this.localPlayer.setPosition(x, y);
      // Update animation based on state
      const animKey = animation === 'running' ? 'player-run' : 'player-idle';
      if (!this.localPlayer.anims.currentAnim || this.localPlayer.anims.currentAnim.key !== animKey) {
        this.localPlayer.play(animKey);
      }
      // Note: Direction is handled in the update loop based on velocity
    }
  }

  update(time: number): void {
    if (!this.localPlayer) return;

    // Handle player movement
    let velocityX = 0;
    let velocityY = 0;
    let isMoving = false;

    // Horizontal movement
    if (this.cursors.left.isDown || this.wasd.A.isDown) {
      velocityX = -this.playerSpeed;
      isMoving = true;
    } else if (this.cursors.right.isDown || this.wasd.D.isDown) {
      velocityX = this.playerSpeed;
      isMoving = true;
    }

    // Vertical movement
    if (this.cursors.up.isDown || this.wasd.W.isDown) {
      velocityY = -this.playerSpeed;
      isMoving = true;
    } else if (this.cursors.down.isDown || this.wasd.S.isDown) {
      velocityY = this.playerSpeed;
      isMoving = true;
    }

    // Apply velocity to player
    this.localPlayer.setVelocity(velocityX, velocityY);

    // Get mouse position in world coordinates
    const pointer = this.input.activePointer;
    this.mouseWorldX = pointer.worldX;
    this.mouseWorldY = pointer.worldY;

    // Flip sprite based on mouse position relative to player
    if (this.mouseWorldX < this.localPlayer.x) {
      this.localPlayer.setFlipX(true); // Mouse is to the left
    } else {
      this.localPlayer.setFlipX(false); // Mouse is to the right
    }

    // Update animation based on movement
    const animation = isMoving ? 'running' : 'idle';
    const animKey = isMoving ? 'player-run' : 'player-idle';

    // Only change animation if it's different from current
    if (!this.localPlayer.anims.currentAnim || this.localPlayer.anims.currentAnim.key !== animKey) {
      this.localPlayer.play(animKey);
    }

    // Update name text position
    if (this.localPlayer.nameText) {
      this.localPlayer.nameText.setPosition(
        this.localPlayer.x,
        this.localPlayer.y - 20
      );
    }

    // Update gun sprite position and rotation - use body position for smoother movement
    if (this.localPlayer.gunSprite && this.localPlayer.body) {
      // Use the body's center position for more accurate positioning during movement
      const bodyCenter = this.localPlayer.body.center;
      this.localPlayer.gunSprite.setPosition(
        bodyCenter.x,
        bodyCenter.y
      );

      // Calculate angle to mouse
      const angle = Phaser.Math.Angle.Between(
        bodyCenter.x,
        bodyCenter.y,
        this.mouseWorldX,
        this.mouseWorldY
      );

      // Check if mouse is on the left or right side of the player
      const isMouseLeft = this.mouseWorldX < bodyCenter.x;

      // Flip gun horizontally if aiming to the left
      if (isMouseLeft) {
        // Aiming left - flip the gun horizontally
        this.localPlayer.gunSprite.setFlipX(true);
        // When flipped, add PI to point correctly
        this.localPlayer.gunSprite.setRotation(angle + Math.PI);
      } else {
        // Aiming right - normal rotation
        this.localPlayer.gunSprite.setFlipX(false);
        this.localPlayer.gunSprite.setRotation(angle);
      }
    }

    // Send position updates to server (throttled)
    if (time - this.lastUpdateTime > this.updateThrottle) {
      const currentX = this.localPlayer.x;
      const currentY = this.localPlayer.y;
      const direction = this.localPlayer.flipX ? 'left' : 'right';

      // Check if position changed significantly (more than 1 pixel) or direction changed
      if (Math.abs(currentX - this.lastPositionUpdate.x) > 1 ||
          Math.abs(currentY - this.lastPositionUpdate.y) > 1 ||
          animation !== this.lastPositionUpdate.animation ||
          direction !== this.lastPositionUpdate.direction) {

        this.sendPositionUpdate(currentX, currentY, animation, direction);
        this.lastPositionUpdate = { x: currentX, y: currentY, animation, direction };
        this.lastUpdateTime = time;
      }
    }
  }

  sendPositionUpdate(x: number, y: number, animation: string, direction: string = 'right'): void {
    if (this.ws && this.ws.send) {
      this.ws.send('updatePosition', {
        x: x,
        y: y,
        animation: animation,
        direction: direction
      });
    }
  }

  adjustZoom(delta: number): void {
    // Use integer zoom levels for pixel-perfect rendering
    this.currentZoom = Phaser.Math.Clamp(this.currentZoom + delta, 2, 5);
    this.cameras.main.setZoom(Math.floor(this.currentZoom));
  }
}