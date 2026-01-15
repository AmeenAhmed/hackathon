import Phaser from 'phaser';

export default class UIScene extends Phaser.Scene {
  public gunText!: Phaser.GameObjects.Text;
  public playerCountText!: Phaser.GameObjects.Text;
  public ammoText!: Phaser.GameObjects.Text;
  public waitingText!: Phaser.GameObjects.Text;

  constructor() {
    super({ key: 'UIScene' });
  }

  create(): void {
    // Get the main scene to access room code and player ID
    const mainScene = this.scene.get('MainScene') as any;

    // Reset camera to ensure it doesn't follow anything
    this.cameras.main.setScroll(0, 0);
    this.cameras.main.setZoom(1);

    // Create a container for all UI elements
    const uiContainer = this.add.container(0, 0);

    // Create UI background
    const bg = this.add.rectangle(100, 75, 200, 150, 0x000000, 0.7);
    uiContainer.add(bg);

    // Room code
    const roomText = this.add.text(10, 10, `Room: ${mainScene.roomCode}`, {
      fontSize: '18px',
      color: '#ffffff',
      fontStyle: 'bold'
    });
    uiContainer.add(roomText);

    // Gun text
    this.gunText = this.add.text(10, 40, 'Gun: Pistol', {
      fontSize: '16px',
      color: '#ffff00'
    });
    uiContainer.add(this.gunText);

    // Player count
    this.playerCountText = this.add.text(10, 70, 'Players: 1', {
      fontSize: '16px',
      color: '#00ff00'
    });
    uiContainer.add(this.playerCountText);

    // Ammo count
    this.ammoText = this.add.text(10, 100, 'Ammo: 30/30', {
      fontSize: '16px',
      color: '#ff9900'
    });
    uiContainer.add(this.ammoText);

    // Controls
    const controlsText = this.add.text(10, 130, 'WASD: Move | 1-3: Guns', {
      fontSize: '14px',
      color: '#ffffff'
    });
    uiContainer.add(controlsText);

    // Set the container to be fixed on screen
    uiContainer.setScrollFactor(0);
    uiContainer.setDepth(1000);

    // Create waiting text (at top of screen like HUD)
    const centerX = this.cameras.main.width / 2;
    const topY = 40; // Position at top of screen

    // Create background for waiting text (wider to cover full text)
    const waitingBg = this.add.rectangle(centerX, topY, 720, 40, 0x000000, 0.8);
    waitingBg.setScrollFactor(0);
    waitingBg.setDepth(1000);

    this.waitingText = this.add.text(centerX, topY, '⚠ Waiting for game to start • Move with WASD • Shooting disabled', {
      fontSize: '18px',
      color: '#ffcc00',
      fontStyle: 'bold',
      align: 'center'
    });
    this.waitingText.setOrigin(0.5, 0.5);
    this.waitingText.setScrollFactor(0);
    this.waitingText.setDepth(1001);
    this.waitingText.setVisible(true); // Initially visible

    // Add pulsing animation to the waiting text
    this.tweens.add({
      targets: this.waitingText,
      alpha: 0.7,
      duration: 1000,
      ease: 'Power2',
      yoyo: true,
      repeat: -1
    });

    // Store background reference to hide it later
    (this as any).waitingBg = waitingBg;
  }

  setWaitingVisible(visible: boolean): void {
    if (this.waitingText) {
      this.waitingText.setVisible(visible);
    }
    // Also hide/show the background
    if ((this as any).waitingBg) {
      (this as any).waitingBg.setVisible(visible);
    }
  }
}