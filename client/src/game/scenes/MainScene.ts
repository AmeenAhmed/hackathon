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
  gunRotation?: number;
  gunFlipped?: boolean;
  currentGun?: number;
}

interface PlayerSprite extends Phaser.Physics.Arcade.Sprite {
  playerId?: string;
  nameText?: Phaser.GameObjects.Text;
  gunSprite?: Phaser.GameObjects.Sprite;
  isDead?: boolean;
  isProtected?: boolean;
}

export default class MainScene extends Phaser.Scene {
  public roomCode!: string;
  public playerId!: string;
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
  private updateThrottle: number = 20; // milliseconds - for smoother updates with 60Hz server
  private lastUpdateTime: number = 0;
  private tilemap!: Phaser.Tilemaps.Tilemap;
  private terrainLayer!: Phaser.Tilemaps.TilemapLayer;
  private worldWidth: number = 1500;  // Reduced from 3000 for performance
  private worldHeight: number = 1500; // Reduced from 3000 for performance
  private currentZoom: number = 3;
  private zoomKeys!: {
    plus: Phaser.Input.Keyboard.Key;
    minus: Phaser.Input.Keyboard.Key;
  };
  private mouseWorldX: number = 0;
  private mouseWorldY: number = 0;

  // Bullet system
  private bullets!: Phaser.Physics.Arcade.Group;
  private bulletSpeed: number = 400;
  private currentGun: number = 0; // 0: pistol, 1: shotgun, 2: uzi
  private lastFireTime: number = 0;
  private fireRates: { [key: number]: number } = {
    0: 1000,  // Pistol: 1 shot per second
    1: 800,   // Shotgun: slightly slower
    2: 100    // Uzi: rapid fire for burst
  };
  private uziBurstCount: number = 0;
  private isMouseDown: boolean = false;
  private bulletCheckCounter: number = 0;
  private activeBullets: Map<string, any> = new Map(); // Track our bullets by ID
  private bulletIdCounter: number = 0;
  private playerHealth: Map<string, number> = new Map(); // Track player health
  private otherPlayerBullets: Map<string, Phaser.GameObjects.Sprite> = new Map(); // Track other players' bullets
  private isDead: boolean = false;
  private isInAmmoQuiz: boolean = false;
  private gameStarted: boolean = false; // Track if game has started

  // Kill tracking
  private killCount: number = 0;
  private killStreak: number = 0;
  private lastKillTime: number = 0;
  private totalKills: number = 0;
  private isFirstBlood: boolean = false;

  // Score tracking
  private correctAnswers: number = 0;

  // Ammo system
  private ammo: { [key: number]: number } = {
    0: 6,   // Pistol ammo
    1: 4,   // Shotgun ammo
    2: 15   // Uzi ammo
  };
  private maxAmmo: { [key: number]: number } = {
    0: 6,   // Pistol max ammo
    1: 8,   // Shotgun max ammo
    2: 30   // Uzi max ammo
  };
  private isReloading: boolean = false;

  constructor(data: SceneData) {
    super({ key: 'MainScene' });
    if (data) {
      this.roomCode = data.roomCode;
      this.playerId = data.playerId;
      this.ws = data.ws;
      // console.log('MainScene created with data:', { roomCode: this.roomCode, playerId: this.playerId });
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
    // console.log('MainScene create() called');

    // Define tilemap dimensions
    const tileSize = 16;
    const mapWidth = Math.ceil(this.worldWidth / tileSize);
    const mapHeight = Math.ceil(this.worldHeight / tileSize);

    // console.log(`Creating tilemap for ${this.worldWidth}x${this.worldHeight} world (${mapWidth}x${mapHeight} tiles)`);

    // Set world bounds larger than viewport
    this.physics.world.setBounds(0, 0, this.worldWidth, this.worldHeight);

    // Create the tilemap
    this.createTilemap(mapWidth, mapHeight, tileSize);

    // Create player animations
    this.createPlayerAnimations();

    // Create bullet group before creating player
    this.createBulletGroup();

    // Listen for bullets hitting world bounds
    this.physics.world.on('worldbounds', (body: Phaser.Physics.Arcade.Body) => {
      // Check if the body belongs to a bullet
      const gameObject = body.gameObject as any;
      if (gameObject && this.bullets.contains(gameObject)) {
        this.destroyBullet(gameObject);
      }
    });

    // Create local player (always visible, even during waiting)
    this.createLocalPlayer();

    // Setup camera to follow player in the larger world
    this.cameras.main.setBounds(0, 0, this.worldWidth, this.worldHeight);
    this.cameras.main.startFollow(this.localPlayer, true, 1, 1); // Instant camera follow (no smoothing for performance)
    this.cameras.main.setZoom(this.currentZoom); // Zoom in closer to the player
    // this.cameras.main.setRoundPixels(true); // Disabled for performance


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

    // Add gun switching with number keys
    this.input.keyboard!.on('keydown-ONE', () => this.switchGun(0));
    this.input.keyboard!.on('keydown-TWO', () => this.switchGun(1));
    this.input.keyboard!.on('keydown-THREE', () => this.switchGun(2));

    // Start the UI scene in parallel
    this.scene.launch('UIScene');

    // Add crosshair cursor
    this.input.setDefaultCursor('crosshair');

    // Setup mouse click handling for firing
    this.input.on('pointerdown', () => {
      this.isMouseDown = true;
    });

    this.input.on('pointerup', () => {
      this.isMouseDown = false;
      this.uziBurstCount = 0; // Reset burst count when releasing mouse
    });

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

  createBulletGroup(): void {
    // Create bullet group with physics
    this.bullets = this.physics.add.group({
      defaultKey: 'bullets',
      maxSize: 100, // Pool size for performance
      runChildUpdate: true
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

    // Enable culling for performance - only render visible tiles
    this.terrainLayer.setCullPadding(2, 2);

    // console.log(`Tilemap created: ${mapWidth}x${mapHeight} tiles (${tileSize}px each) = ${mapWidth * tileSize}x${mapHeight * tileSize}px total`);
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
    this.localPlayer.setVisible(true); // Ensure player is visible

    // Start with idle animation
    this.localPlayer.play('player-idle');

    // Set player color if available from store
    const playerStore = (window as any).playerStore;
    if (playerStore && playerStore.color) {
      this.localPlayer.setTint(parseInt(playerStore.color.replace('#', '0x')));
    }

    // Add player name text - use actual player name from store
    const playerName = playerStore?.name || 'You';

    const nameText = this.add.text(0, -20, playerName, {
      fontSize: '10px',
      color: '#ffffff',
      backgroundColor: '#000000',
      padding: { x: 2, y: 1 }
    });
    nameText.setOrigin(0.5, 0.5);
    nameText.setDepth(11); // Above player
    nameText.setVisible(true); // Ensure name is visible
    this.localPlayer.nameText = nameText;

    // Add gun sprite (use frame 0, first gun)
    const gunSprite = this.add.sprite(startX, startY, 'guns', 0);
    gunSprite.setScale(1);
    gunSprite.setDepth(15); // Higher depth to ensure it's always on top
    gunSprite.setOrigin(0.5, 0.5);
    gunSprite.setVisible(true); // Ensure gun is visible
    this.localPlayer.gunSprite = gunSprite;

    // Initialize ammo UI
    this.updateAmmoUI();
  }

  setupWebSocketListeners(): void {
    // Listen for game state updates
    if (this.ws) {
      this.ws.on('gameUpdate', (data: any) => {
        // Only update if scene is active
        if (this.scene.isActive()) {
          // console.log('Received gameUpdate with players:', Object.keys(data.gameState?.players || {}));
          this.updateGameState(data.gameState);
        }
      });

      this.ws.on('initialState', (data: any) => {
        // Only update if scene is active
        if (this.scene.isActive()) {
          // console.log('Received initialState with players:', Object.keys(data.gameState?.players || {}));
          this.updateGameState(data.gameState);

          // Check if game has already started
          if (data.gameState?.gamePhase === 'playing') {
            this.gameStarted = true;
            const uiScene = this.scene.get('UIScene') as any;
            if (uiScene && uiScene.setWaitingVisible) {
              uiScene.setWaitingVisible(false);
            }
          }
        }
      });

      // Request initial state when joining (important for seeing existing players)
      // console.log('Requesting initial game state from server...');
      this.ws.send('getState', {});

      // Listen for game started event
      this.ws.on('gameStarted', (data: any) => {
        if (this.scene.isActive()) {
          this.gameStarted = true;
          const uiScene = this.scene.get('UIScene') as any;
          if (uiScene && uiScene.setWaitingVisible) {
            uiScene.setWaitingVisible(false);
          }
        }
      });

      // Listen for bullet spawns from other players
      this.ws.on('bulletSpawn', (data: any) => {
        if (this.scene.isActive() && data.ownerId !== this.playerId) {
          this.spawnOtherPlayerBullet(data);
        }
      });

      // Listen for bullet destroys
      this.ws.on('bulletDestroy', (data: any) => {
        if (this.scene.isActive()) {
          this.removeOtherPlayerBullet(data.bulletId);
        }
      });

      // Listen for player hits (to show visual effects)
      this.ws.on('playerHit', (data: any) => {
        if (this.scene.isActive()) {
          this.handleRemotePlayerHit(data);
        }
      });

      // Listen for player deaths
      this.ws.on('playerDeath', (data: any) => {
        if (this.scene.isActive()) {
          this.handleRemotePlayerDeath(data);
        }
      });

      // Listen for player respawns
      this.ws.on('playerRespawn', (data: any) => {
        if (this.scene.isActive()) {
          this.handleRemotePlayerRespawn(data);
        }
      });
    }
  }

  updateGameState(gameState: any): void {
    if (!gameState) return;

    // Check if game phase has changed
    if (gameState.gamePhase === 'playing' && !this.gameStarted) {
      this.gameStarted = true;
      const uiScene = this.scene.get('UIScene') as any;
      if (uiScene && uiScene.setWaitingVisible) {
        uiScene.setWaitingVisible(false);
      }
    }

    if (!gameState.players) return;

    // Make sure the scene is ready and physics is initialized
    if (!this.physics || !this.physics.world) {
      // console.warn('Physics not ready, skipping game state update');
      return;
    }

    // Debug: Log full game state
    // console.log('Full game state received:', JSON.stringify(gameState));

    // Update player count display
    const playerCount = Object.keys(gameState.players).length;
    const uiScene = this.scene.get('UIScene') as any;
    if (uiScene && uiScene.playerCountText) {
      uiScene.playerCountText.setText(`Players: ${playerCount}`);
    }

    // Update other players
    for (const [playerId, playerData] of Object.entries(gameState.players)) {
      if (playerId !== this.playerId) {
        // console.log(`Processing player ${playerId}:`, playerData);
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
      // console.log('Creating new player:', playerId, 'with full data:', JSON.stringify(playerData));

      // Create new player sprite - use same sprites as local player
      sprite = this.physics.add.sprite(
        playerData.x || this.worldWidth / 2,
        playerData.y || this.worldHeight / 2,
        'player-idle'
      ) as PlayerSprite;

      // Check if sprite was created successfully
      if (!sprite) {
        // console.error('Failed to create sprite for player:', playerId);
        return;
      }

      // Make sure the sprite has a physics body and set it up like local player
      sprite.setCollideWorldBounds(true);
      sprite.setScale(1);
      sprite.playerId = playerId;
      sprite.isDead = false; // Initialize as alive
      sprite.setDepth(10); // Same depth as local player

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

      // Add player name - use their actual name
      const displayName = playerData.name || `Player ${playerId.substring(0, 8)}`;
      // console.log('Creating name text for player:', playerId, 'with name:', displayName);

      const nameText = this.add.text(
        playerData.x || this.worldWidth / 2,
        (playerData.y || this.worldHeight / 2) - 20,
        displayName, {
        fontSize: '12px',
        color: '#00ff00',  // Green color to distinguish from local player
        backgroundColor: '#000000',
        padding: { x: 3, y: 2 }
      });
      nameText.setOrigin(0.5, 0.5);
      nameText.setDepth(11); // Above player
      sprite.nameText = nameText;

      // Add gun sprite for other players
      const gunSprite = this.add.sprite(
        playerData.x || this.worldWidth / 2,
        playerData.y || this.worldHeight / 2,
        'guns',
        0
      );
      gunSprite.setScale(1);
      gunSprite.setDepth(15); // Higher depth to ensure it's always on top
      gunSprite.setOrigin(0.5, 0.5);
      sprite.gunSprite = gunSprite;

      this.otherPlayers.set(playerId, sprite);
    } else {
      // Debug log position updates
      // console.log('Updating player:', playerId, 'to position:', playerData.x, playerData.y);

      // Direct position update - no tween for instant response
      sprite.setPosition(playerData.x, playerData.y);

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

      // Update name position and text
      if (sprite.nameText) {
        // Update the name text in case it changed
        const displayName = playerData.name || `Player ${playerId.substring(0, 8)}`;
        sprite.nameText.setText(displayName);

        // Update position directly
        sprite.nameText.setPosition(playerData.x, playerData.y - 20);
      }

      // Update gun position, rotation and flip
      if (sprite.gunSprite) {
        sprite.gunSprite.setPosition(playerData.x, playerData.y);

        // Apply gun rotation and flip from server data
        if (playerData.gunRotation !== undefined) {
          sprite.gunSprite.setRotation(playerData.gunRotation);
        }
        if (playerData.gunFlipped !== undefined) {
          sprite.gunSprite.setFlipX(playerData.gunFlipped);
        }
        if (playerData.currentGun !== undefined) {
          sprite.gunSprite.setFrame(playerData.currentGun);
        }
      }

      // Handle protection state (in quiz or respawning)
      if (playerData.isProtected !== undefined) {
        sprite.isProtected = playerData.isProtected;

        // Show semi-transparent if protected
        if (playerData.isProtected) {
          sprite.setAlpha(0.5);
          sprite.nameText?.setAlpha(0.5);
          sprite.gunSprite?.setAlpha(0.5);
        } else {
          sprite.setAlpha(1);
          sprite.nameText?.setAlpha(1);
          sprite.gunSprite?.setAlpha(1);
        }
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
    if (!this.localPlayer || !this.localPlayer.body) return;

    // Don't process input if dead and showing quiz
    if (this.isDead) return;

    // Movement is allowed even during waiting state
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

    // Handle firing (only if game has started)
    if (this.isMouseDown && this.gameStarted) {
      this.handleFiring(time);
    }

    // Update bullets - only check every 3rd frame for performance
    this.bulletCheckCounter++;
    if (this.bulletCheckCounter >= 3) {
      this.bulletCheckCounter = 0;

      // Check our active bullets for collisions and distance
      this.bullets.children.entries.forEach((bullet: any) => {
        if (!bullet.active || !bullet.ownerId || bullet.ownerId !== this.playerId) return;

        // Check distance first
        const distance = Phaser.Math.Distance.Between(
          bullet.startX,
          bullet.startY,
          bullet.x,
          bullet.y
        );

        // Destroy bullets that have traveled too far
        if (distance > 1500) {
          this.destroyBullet(bullet);
          return;
        }

        // Check collision with other players
        for (const [playerId, otherPlayer] of this.otherPlayers) {
          // Skip dead or protected players - they're in quiz mode and shouldn't have collision
          if (otherPlayer.isDead || otherPlayer.isProtected) {
            continue;
          }

          if (this.checkBulletPlayerCollision(bullet, otherPlayer)) {
            // Handle hit
            this.handleBulletHit(bullet, playerId, otherPlayer);
            break;
          }
        }
      });
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
      // Calculate gun rotation for sending to server
      const bodyCenter = this.localPlayer.body.center;
      const angle = Phaser.Math.Angle.Between(
        bodyCenter.x,
        bodyCenter.y,
        this.mouseWorldX,
        this.mouseWorldY
      );
      const gunFlipped = this.mouseWorldX < bodyCenter.x;
      const gunRotation = gunFlipped ? angle + Math.PI : angle;

      this.ws.send('updatePosition', {
        x: x,
        y: y,
        animation: animation,
        direction: direction,
        gunRotation: gunRotation,
        gunFlipped: gunFlipped,
        currentGun: this.currentGun,
        isProtected: this.isDead || this.isInAmmoQuiz  // Add protection state
      });
    }
  }

  adjustZoom(delta: number): void {
    // Use integer zoom levels for pixel-perfect rendering
    this.currentZoom = Phaser.Math.Clamp(this.currentZoom + delta, 2, 5);
    this.cameras.main.setZoom(Math.floor(this.currentZoom));
  }

  handleFiring(time: number): void {
    // Don't allow firing if game hasn't started
    if (!this.gameStarted) {
      return;
    }

    // Check if we're reloading
    if (this.isReloading) {
      return;
    }

    // Check if current gun has ammo
    if (this.ammo[this.currentGun] <= 0) {
      // Check if ANY gun has ammo
      const hasAnyAmmo = this.ammo[0] > 0 || this.ammo[1] > 0 || this.ammo[2] > 0;

      if (!hasAnyAmmo) {
        // All guns out of ammo, trigger reload quiz for all
        this.triggerAmmoQuiz();
      } else {
        // Just this gun is out, show a message and try to auto-switch
        // Try to switch to a gun with ammo
        for (let i = 0; i < 3; i++) {
          if (this.ammo[i] > 0 && i !== this.currentGun) {
            this.switchGun(i);
            break;
          }
        }
      }
      return;
    }

    // Check fire rate cooldown
    if (time - this.lastFireTime < this.fireRates[this.currentGun]) {
      return;
    }

    // Use the player sprite's actual position for consistent bullet spawning
    const playerX = this.localPlayer.x;
    const playerY = this.localPlayer.y;

    // Calculate base angle to mouse
    const baseAngle = Phaser.Math.Angle.Between(
      playerX,
      playerY,
      this.mouseWorldX,
      this.mouseWorldY
    );

    switch (this.currentGun) {
      case 0: // Pistol - single shot
        this.fireBullet(playerX, playerY, baseAngle);
        // Play pistol sound
        this.sound.play('pistol-fire', { volume: 0.5 });
        this.ammo[0]--;
        this.updateAmmoUI();
        this.lastFireTime = time;
        break;

      case 1: // Shotgun - 3 bullets in cone
        const spreadAngle = Phaser.Math.DegToRad(15); // 15 degree spread
        this.fireBullet(playerX, playerY, baseAngle - spreadAngle);
        this.fireBullet(playerX, playerY, baseAngle);
        this.fireBullet(playerX, playerY, baseAngle + spreadAngle);
        // Play shotgun sound
        this.sound.play('shotgun-fire', { volume: 0.6 });
        this.ammo[1]--;
        this.updateAmmoUI();
        this.lastFireTime = time;
        break;

      case 2: // Uzi - burst of 3 bullets
        if (this.uziBurstCount < 3) {
          this.fireBullet(playerX, playerY, baseAngle);
          // Play uzi/pistol sound (slightly quieter for rapid fire)
          this.sound.play('pistol-fire', { volume: 0.3 });
          this.uziBurstCount++;
          this.ammo[2]--;
          this.updateAmmoUI();
          this.lastFireTime = time;
        } else {
          // After 3 bullets, need longer cooldown before next burst
          if (time - this.lastFireTime > 300) {
            this.uziBurstCount = 0;
          }
        }
        break;
    }
  }

  fireBullet(x: number, y: number, angle: number): void {
    // Don't create bullets if game hasn't started
    if (!this.gameStarted) {
      return;
    }

    // Get or create a bullet from the pool
    let bullet = this.bullets.get(x, y, 'bullets', 0) as any;

    if (bullet) {
      bullet.setActive(true);
      bullet.setVisible(true);
      bullet.body.enable = true;
      bullet.setScale(1);
      bullet.setDepth(9); // Below player (player is at depth 10)

      // Generate unique bullet ID
      const bulletId = `${this.playerId}_${this.bulletIdCounter++}`;
      bullet.bulletId = bulletId;
      bullet.ownerId = this.playerId;
      bullet.gunType = this.currentGun;

      // Store starting position for distance calculation
      bullet.startX = x;
      bullet.startY = y;

      // Set bullet to collide with world bounds and be destroyed on collision
      bullet.body.setCollideWorldBounds(true);
      bullet.body.onWorldBounds = true;

      // Set bullet velocity based on angle
      const velocityX = Math.cos(angle) * this.bulletSpeed;
      const velocityY = Math.sin(angle) * this.bulletSpeed;
      bullet.setVelocity(velocityX, velocityY);

      // Rotate bullet to match firing angle
      bullet.setRotation(angle);

      // Track this bullet
      this.activeBullets.set(bulletId, bullet);

      // Add camera shake effect
      const shakeIntensity = this.currentGun === 1 ? 1.5 : 1; // Shotgun has slightly stronger shake
      const shakeDuration = this.currentGun === 1 ? 50 : 30; // Shotgun shakes slightly longer
      this.cameras.main.shake(shakeDuration, shakeIntensity * 0.001);

      // Send bullet spawn message to server
      this.ws.send('bulletSpawn', {
        bulletId: bulletId,
        x: x,
        y: y,
        velocityX: velocityX,
        velocityY: velocityY,
        angle: angle,
        gunType: this.currentGun
      });
    }
  }

  switchGun(gunIndex: number): void {
    this.currentGun = gunIndex;
    this.uziBurstCount = 0; // Reset burst count when switching guns

    // Update gun sprite frame
    if (this.localPlayer.gunSprite) {
      this.localPlayer.gunSprite.setFrame(gunIndex);
    }

    // Update gun text and ammo
    const gunNames = ['Pistol', 'Shotgun', 'Uzi'];
    const uiScene = this.scene.get('UIScene') as any;
    if (uiScene && uiScene.gunText) {
      uiScene.gunText.setText(`Gun: ${gunNames[gunIndex]}`);
    }
    this.updateAmmoUI();
  }

  checkBulletPlayerCollision(bullet: any, player: PlayerSprite): boolean {
    // Simple rectangular collision detection
    const bulletBounds = bullet.getBounds();
    const playerBounds = player.getBounds();
    return Phaser.Geom.Rectangle.Overlaps(bulletBounds, playerBounds);
  }

  destroyBullet(bullet: any): void {
    if (bullet.bulletId) {
      this.activeBullets.delete(bullet.bulletId);
      // Send bullet destroy message to server
      this.ws.send('bulletDestroy', {
        bulletId: bullet.bulletId
      });
    }
    this.bullets.killAndHide(bullet);
    bullet.body.enable = false;
  }

  handleBulletHit(bullet: any, targetPlayerId: string, targetPlayer: PlayerSprite): void {
    // Calculate damage based on gun type
    const damage = bullet.gunType === 1 ? 15 : 25; // Shotgun does less damage per pellet

    // Get or initialize target player health
    let health = this.playerHealth.get(targetPlayerId) ?? 100;
    health -= damage;
    this.playerHealth.set(targetPlayerId, health);

    // Flash white effect for hit
    this.flashWhite(targetPlayer);

    // Send hit message to server
    this.ws.send('playerHit', {
      bulletId: bullet.bulletId,
      targetPlayerId: targetPlayerId,
      damage: damage,
      health: health,
      isDead: health <= 0
    });

    // Destroy the bullet
    this.destroyBullet(bullet);

    // If target died, send death message and handle kill
    if (health <= 0) {
      this.ws.send('playerDeath', {
        playerId: targetPlayerId
      });

      // Handle kill tracking and announcements
      this.handleKill();
    }
  }

  flashWhite(sprite: PlayerSprite): void {
    // Apply white silhouette effect (fills entire sprite with white)
    sprite.setTintFill(0xffffff);

    // Remove tint after 100ms
    this.time.delayedCall(100, () => {
      // Clear the tint fill and restore original tint
      sprite.clearTint();
      const playerData = this.getPlayerData(sprite.playerId!);
      if (playerData?.color) {
        sprite.setTint(parseInt(playerData.color.replace('#', '0x')));
      }
    });
  }

  getPlayerData(playerId: string): Player | undefined {
    // This will be populated from game state
    return undefined; // Placeholder for now
  }


  spawnOtherPlayerBullet(data: any): void {
    // Create bullet for other player
    const bullet = this.add.sprite(data.x, data.y, 'bullets', 0);
    bullet.setScale(1);
    bullet.setDepth(9);
    bullet.setRotation(data.angle);

    // Store in tracking map
    this.otherPlayerBullets.set(data.bulletId, bullet);

    // Create tween for bullet movement
    const distance = 1500;
    const duration = distance / this.bulletSpeed * 1000;
    const endX = data.x + Math.cos(data.angle) * distance;
    const endY = data.y + Math.sin(data.angle) * distance;

    this.tweens.add({
      targets: bullet,
      x: endX,
      y: endY,
      duration: duration,
      ease: 'Linear',
      onComplete: () => {
        this.removeOtherPlayerBullet(data.bulletId);
      }
    });
  }

  removeOtherPlayerBullet(bulletId: string): void {
    const bullet = this.otherPlayerBullets.get(bulletId);
    if (bullet) {
      bullet.destroy();
      this.otherPlayerBullets.delete(bulletId);
    }
  }

  handleRemotePlayerHit(data: any): void {
    // Update health tracking
    if (data.targetPlayerId && data.health !== undefined) {
      this.playerHealth.set(data.targetPlayerId, data.health);
    }

    // Flash the hit player
    if (data.targetPlayerId === this.playerId) {
      // Local player was hit
      this.flashWhite(this.localPlayer);
    } else {
      // Other player was hit
      const otherPlayer = this.otherPlayers.get(data.targetPlayerId);
      if (otherPlayer) {
        this.flashWhite(otherPlayer);
      }
    }
  }

  handleRemotePlayerDeath(data: any): void {
    if (data.playerId === this.playerId) {
      // Local player died - show quiz
      this.localPlayer.setVisible(false);
      this.localPlayer.nameText?.setVisible(false);
      this.localPlayer.gunSprite?.setVisible(false);

      // Mark as dead
      this.isDead = true;

      // Reset kill streak on death
      this.killStreak = 0;

      // Show the Vue quiz component
      if ((window as any).showDeathQuiz) {
        (window as any).showDeathQuiz();
      }

      // Set up quiz completion callback
      (window as any).onQuizComplete = () => {
        // Respawn the player
        const spawnX = this.worldWidth / 2;
        const spawnY = this.worldHeight / 2;

        this.localPlayer.setPosition(spawnX, spawnY);
        this.localPlayer.setVisible(true);
        this.localPlayer.nameText?.setVisible(true);
        this.localPlayer.gunSprite?.setVisible(true);

        this.isDead = false;
        this.playerHealth.set(this.playerId, 100);

        // Reset ammo on respawn (to starting values, not max)
        this.ammo[0] = 6;   // Pistol starting ammo
        this.ammo[1] = 4;   // Shotgun starting ammo
        this.ammo[2] = 15;  // Uzi starting ammo
        this.updateAmmoUI();

        // Send respawn message
        this.ws.send('playerRespawn', {
          playerId: this.playerId,
          x: spawnX,
          y: spawnY
        });
      };
    } else {
      // Other player died
      const otherPlayer = this.otherPlayers.get(data.playerId);
      if (otherPlayer) {
        otherPlayer.setVisible(false);
        otherPlayer.nameText?.setVisible(false);
        otherPlayer.gunSprite?.setVisible(false);
        // Mark other player as dead
        otherPlayer.isDead = true;
      }
    }
  }

  handleKill(): void {
    const currentTime = this.time.now;

    // Update kill counts
    this.totalKills++;

    // Send kill update to server for score calculation
    this.sendKillUpdate();

    // Check if it's a streak (within 5 seconds of last kill)
    if (currentTime - this.lastKillTime < 5000) {
      this.killStreak++;
    } else {
      this.killStreak = 1;
    }
    this.lastKillTime = currentTime;

    // Play appropriate announcement based on kill streak and total kills
    if (this.totalKills === 1 && !this.isFirstBlood) {
      this.isFirstBlood = true;
      this.sound.play('firstblood', { volume: 0.7 });
    } else if (this.killStreak === 2) {
      this.sound.play('doublekill', { volume: 0.7 });
    } else if (this.killStreak === 3) {
      this.sound.play('triplekill', { volume: 0.7 });
    } else if (this.killStreak === 5) {
      this.sound.play('killingspree', { volume: 0.7 });
    } else if (this.killStreak === 7) {
      this.sound.play('rampage', { volume: 0.7 });
    } else if (this.killStreak === 9) {
      this.sound.play('dominating', { volume: 0.7 });
    } else if (this.killStreak === 11) {
      this.sound.play('megakill', { volume: 0.7 });
    } else if (this.killStreak === 13) {
      this.sound.play('ownage', { volume: 0.7 });
    } else if (this.killStreak === 15) {
      this.sound.play('massacre', { volume: 0.7 });
    } else if (this.killStreak === 17) {
      this.sound.play('carnage', { volume: 0.7 });
    } else if (this.killStreak === 19) {
      this.sound.play('mayhem', { volume: 0.7 });
    } else if (this.killStreak >= 20) {
      this.sound.play('godlike', { volume: 0.7 });
    }

    // Display kill streak text (optional visual feedback)
    if (this.killStreak >= 2) {
      const streakText = this.add.text(
        this.cameras.main.width / 2,
        100,
        `${this.getStreakName(this.killStreak)}!`,
        {
          fontSize: '32px',
          color: '#ffff00',
          fontStyle: 'bold'
        }
      );
      streakText.setOrigin(0.5, 0.5);
      streakText.setScrollFactor(0);
      streakText.setDepth(100);

      // Fade out and destroy after 2 seconds
      this.tweens.add({
        targets: streakText,
        alpha: 0,
        duration: 2000,
        ease: 'Power2',
        onComplete: () => {
          streakText.destroy();
        }
      });
    }
  }

  getStreakName(streak: number): string {
    if (streak === 2) return 'DOUBLE KILL';
    if (streak === 3) return 'TRIPLE KILL';
    if (streak === 5) return 'KILLING SPREE';
    if (streak === 7) return 'RAMPAGE';
    if (streak === 9) return 'DOMINATING';
    if (streak === 11) return 'MEGA KILL';
    if (streak === 13) return 'OWNAGE';
    if (streak === 15) return 'MASSACRE';
    if (streak === 17) return 'CARNAGE';
    if (streak === 19) return 'MAYHEM';
    if (streak >= 20) return 'GODLIKE';
    return '';
  }

  // Send correct answers count to server for score calculation
  sendCorrectAnswers(count: number): void {
    this.correctAnswers += count;
    if (this.ws && this.ws.send) {
      this.ws.send('updateScore', {
        correctAnswers: this.correctAnswers,
        kills: this.totalKills
      });
    }
  }

  // Send kill to server for score calculation
  sendKillUpdate(): void {
    if (this.ws && this.ws.send) {
      this.ws.send('updateScore', {
        correctAnswers: this.correctAnswers,
        kills: this.totalKills
      });
    }
  }

  handleRemotePlayerRespawn(data: any): void {
    if (data.playerId === this.playerId) {
      // Local player respawned
      this.localPlayer.setPosition(data.x, data.y);
      this.localPlayer.setVisible(true);
      this.localPlayer.nameText?.setVisible(true);
      this.localPlayer.gunSprite?.setVisible(true);
      this.playerHealth.set(this.playerId, 100);
    } else {
      // Other player respawned
      const otherPlayer = this.otherPlayers.get(data.playerId);
      if (otherPlayer) {
        otherPlayer.setPosition(data.x, data.y);
        otherPlayer.setVisible(true);
        otherPlayer.nameText?.setVisible(true);
        otherPlayer.gunSprite?.setVisible(true);
        // Clear dead state
        otherPlayer.isDead = false;
        this.playerHealth.set(data.playerId, 100);
      }
    }
  }

  updateAmmoUI(): void {
    const uiScene = this.scene.get('UIScene') as any;
    if (uiScene && uiScene.ammoText) {
      const ammoCount = this.ammo[this.currentGun];
      const maxAmmo = this.maxAmmo[this.currentGun];
      const text = ammoCount > 0 ? `Ammo: ${ammoCount}/${maxAmmo}` : 'Ammo: EMPTY';
      uiScene.ammoText.setText(text);

      // Change color based on ammo status
      if (ammoCount === 0) {
        uiScene.ammoText.setColor('#ff0000'); // Red for empty
      } else if (ammoCount <= maxAmmo / 4) {
        uiScene.ammoText.setColor('#ffaa00'); // Orange for low
      } else {
        uiScene.ammoText.setColor('#ff9900'); // Normal orange
      }
    }
  }

  triggerAmmoQuiz(): void {
    if (this.isReloading) return;

    this.isReloading = true;
    this.isInAmmoQuiz = true;

    // Make player semi-transparent to indicate protection
    this.localPlayer.setAlpha(0.5);
    this.localPlayer.nameText?.setAlpha(0.5);
    this.localPlayer.gunSprite?.setAlpha(0.5);

    // Trigger the ammo quiz via global function
    if ((window as any).showAmmoQuiz) {
      (window as any).showAmmoQuiz();
    }
  }

  reloadAmmo(): void {
    // Called when quiz is completed successfully - reload ALL guns
    this.ammo[0] = this.maxAmmo[0];
    this.ammo[1] = this.maxAmmo[1];
    this.ammo[2] = this.maxAmmo[2];
    this.isReloading = false;
    this.isInAmmoQuiz = false;

    // Restore player opacity
    this.localPlayer.setAlpha(1);
    this.localPlayer.nameText?.setAlpha(1);
    this.localPlayer.gunSprite?.setAlpha(1);

    this.updateAmmoUI();
  }
}