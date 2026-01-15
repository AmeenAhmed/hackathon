import Phaser from 'phaser';

interface SceneData {
  roomCode: string;
  playerId: string;
  ws: any;
}

export default class PreloadScene extends Phaser.Scene {
  private roomCode: string;
  private playerId: string;
  private ws: any;

  constructor(data: SceneData) {
    super({ key: 'PreloadScene' });
    this.roomCode = data.roomCode;
    this.playerId = data.playerId;
    this.ws = data.ws;
    // console.log('PreloadScene created with:', { roomCode: this.roomCode, playerId: this.playerId });
  }

  preload(): void {
    // Create loading text
    const loadingText = this.add.text(
      this.cameras.main.width / 2,
      this.cameras.main.height / 2 - 50,
      'Loading...',
      {
        fontSize: '32px',
        color: '#ffffff'
      }
    );
    loadingText.setOrigin(0.5, 0.5);

    // Create progress bar (centered for 1920x1080)
    const progressBar = this.add.graphics();
    const progressBox = this.add.graphics();
    const boxWidth = 800;
    const boxHeight = 50;
    const boxX = (this.cameras.main.width - boxWidth) / 2;
    const boxY = (this.cameras.main.height / 2) + 50;

    progressBox.fillStyle(0x222222, 0.8);
    progressBox.fillRect(boxX, boxY, boxWidth, boxHeight);

    this.load.on('progress', (value: number) => {
      progressBar.clear();
      progressBar.fillStyle(0xffffff, 1);
      progressBar.fillRect(boxX + 10, boxY + 10, (boxWidth - 20) * value, boxHeight - 20);
    });

    this.load.on('complete', () => {
      progressBar.destroy();
      progressBox.destroy();
      loadingText.destroy();
    });

    // Load terrain spritesheet
    // The spritesheet should have 7 tiles (indices 0-6) with 16x16 pixel tiles
    this.load.spritesheet('terrain-tiles', '/assets/spritesheets/Hackathon-Terrain.png', {
      frameWidth: 16,
      frameHeight: 16,
      margin: 0,
      spacing: 0
    });

    // Load player animation spritesheets (16x16 pixels)
    this.load.spritesheet('player-idle', '/assets/spritesheets/Hackathon-Idle.png', {
      frameWidth: 16,
      frameHeight: 16,
      margin: 0,
      spacing: 0
    });

    this.load.spritesheet('player-run', '/assets/spritesheets/Hackathon-Run.png', {
      frameWidth: 16,
      frameHeight: 16,
      margin: 0,
      spacing: 0
    });

    // Load gun sprites (16x16 pixels)
    this.load.spritesheet('guns', '/assets/spritesheets/Hackathon-Guns.png', {
      frameWidth: 16,
      frameHeight: 16,
      margin: 0,
      spacing: 0
    });

    // Load bullet sprite (16x16 pixels)
    this.load.spritesheet('bullets', '/assets/spritesheets/Hackathon-Bullet.png', {
      frameWidth: 16,
      frameHeight: 16,
      margin: 0,
      spacing: 0
    });

    // Load weapon sounds
    this.load.audio('pistol-fire', '/assets/sounds/pistol.wav');
    this.load.audio('shotgun-fire', '/assets/sounds/shotgun.wav');

    // Load kill streak sounds
    this.load.audio('firstblood', '/assets/sounds/FirstBlood.wav');
    this.load.audio('doublekill', '/assets/sounds/DoubleKill.wav');
    this.load.audio('triplekill', '/assets/sounds/TripleKill.wav');
    this.load.audio('killingspree', '/assets/sounds/KillingSpree.wav');
    this.load.audio('rampage', '/assets/sounds/Rampage.wav');
    this.load.audio('dominating', '/assets/sounds/Dominating.wav');
    this.load.audio('megakill', '/assets/sounds/MegaKill.wav');
    this.load.audio('godlike', '/assets/sounds/Godlike.wav');
    this.load.audio('ownage', '/assets/sounds/Ownage.wav');
    this.load.audio('massacre', '/assets/sounds/Massacre.wav');
    this.load.audio('carnage', '/assets/sounds/Carnage.wav');
    this.load.audio('mayhem', '/assets/sounds/Mayhem.wav');

    // Set textures to use nearest neighbor filtering after load
    this.load.on('filecomplete-spritesheet-terrain-tiles', () => {
      const texture = this.textures.get('terrain-tiles');
      texture.setFilter(Phaser.Textures.FilterMode.NEAREST);
    });

    this.load.on('filecomplete-spritesheet-player-idle', () => {
      const texture = this.textures.get('player-idle');
      texture.setFilter(Phaser.Textures.FilterMode.NEAREST);
    });

    this.load.on('filecomplete-spritesheet-player-run', () => {
      const texture = this.textures.get('player-run');
      texture.setFilter(Phaser.Textures.FilterMode.NEAREST);
    });

    this.load.on('filecomplete-spritesheet-guns', () => {
      const texture = this.textures.get('guns');
      texture.setFilter(Phaser.Textures.FilterMode.NEAREST);
    });

    this.load.on('filecomplete-spritesheet-bullets', () => {
      const texture = this.textures.get('bullets');
      texture.setFilter(Phaser.Textures.FilterMode.NEAREST);
    });
  }

  create(): void {
    // console.log('PreloadScene create() called, starting MainScene');
    // Pass data to the main scene and start it
    this.scene.start('MainScene', {
      roomCode: this.roomCode,
      playerId: this.playerId,
      ws: this.ws
    });
  }
}