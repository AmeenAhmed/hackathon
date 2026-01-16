import Phaser from 'phaser';

export default class UIScene extends Phaser.Scene {
  // HUD elements
  private hudContainer!: Phaser.GameObjects.Container;
  
  // Text elements
  public timerText!: Phaser.GameObjects.Text;
  public gunText!: Phaser.GameObjects.Text;
  public ammoText!: Phaser.GameObjects.Text;
  public killsText!: Phaser.GameObjects.Text;
  public answersText!: Phaser.GameObjects.Text;
  public playerCountText!: Phaser.GameObjects.Text;
  public waitingText!: Phaser.GameObjects.Text;
  public roomText!: Phaser.GameObjects.Text;

  // Controls bar elements (for resize handling)
  private controlsBg!: Phaser.GameObjects.Rectangle;
  private controlsText!: Phaser.GameObjects.Text;

  // Timer value
  private gameTimer: number = 300;
  private timerActive: boolean = false;

  constructor() {
    super({ key: 'UIScene' });
  }

  create(): void {
    const mainScene = this.scene.get('MainScene') as any;
    const screenWidth = this.cameras.main.width;
    const screenHeight = this.cameras.main.height;

    // Reset camera
    this.cameras.main.setScroll(0, 0);
    this.cameras.main.setZoom(1);

    // Create the main HUD container
    this.hudContainer = this.add.container(0, 0);

    // === TOP HEADER BAR ===
    const headerHeight = 50;
    const headerBg = this.add.rectangle(screenWidth / 2, headerHeight / 2, screenWidth, headerHeight, 0x0f172a, 0.92);
    this.hudContainer.add(headerBg);

    // Bottom border line
    const borderLine = this.add.rectangle(screenWidth / 2, headerHeight, screenWidth, 2, 0x8b5cf6, 0.8);
    this.hudContainer.add(borderLine);

    // === LEFT: LOGO + ROOM CODE ===
    const leftX = 16;

    // Logo background
    const logoBg = this.add.rectangle(leftX + 18, headerHeight / 2, 36, 36, 0x8b5cf6, 1);
    logoBg.setStrokeStyle(0);
    this.hudContainer.add(logoBg);

    // Logo "W" text
    const logoW = this.add.text(leftX + 18, headerHeight / 2, 'W', {
      fontSize: '22px',
      color: '#ffffff',
      fontStyle: 'bold',
      fontFamily: 'Arial'
    });
    logoW.setOrigin(0.5, 0.5);
    this.hudContainer.add(logoW);

    // WAYARENA text
    const wayText = this.add.text(leftX + 44, headerHeight / 2 - 8, 'WAY', {
      fontSize: '16px',
      color: '#22d3ee',
      fontStyle: 'bold',
      fontFamily: 'Arial'
    });
    wayText.setOrigin(0, 0.5);
    this.hudContainer.add(wayText);

    const arenaText = this.add.text(leftX + 44, headerHeight / 2 + 8, 'ARENA', {
      fontSize: '16px',
      color: '#f472b6',
      fontStyle: 'bold',
      fontFamily: 'Arial'
    });
    arenaText.setOrigin(0, 0.5);
    this.hudContainer.add(arenaText);

    // Room code badge
    const roomBadgeX = leftX + 120;
    const roomBadgeBg = this.add.rectangle(roomBadgeX + 45, headerHeight / 2, 90, 28, 0x1e293b, 1);
    roomBadgeBg.setStrokeStyle(1, 0x475569);
    this.hudContainer.add(roomBadgeBg);

    this.roomText = this.add.text(roomBadgeX + 45, headerHeight / 2, mainScene.roomCode || 'ROOM', {
      fontSize: '14px',
      color: '#94a3b8',
      fontStyle: 'bold',
      fontFamily: 'monospace'
    });
    this.roomText.setOrigin(0.5, 0.5);
    this.hudContainer.add(this.roomText);

    // === CENTER: TIMER ===
    const centerX = screenWidth / 2;

    // Timer container background
    const timerBg = this.add.rectangle(centerX, headerHeight / 2, 140, 36, 0x1e293b, 1);
    timerBg.setStrokeStyle(2, 0xfbbf24);
    this.hudContainer.add(timerBg);

    // Timer icon
    const timerIcon = this.add.text(centerX - 52, headerHeight / 2, '⏱', {
      fontSize: '20px'
    });
    timerIcon.setOrigin(0.5, 0.5);
    this.hudContainer.add(timerIcon);

    // Timer text
    this.timerText = this.add.text(centerX + 10, headerHeight / 2, '05:00', {
      fontSize: '22px',
      color: '#fbbf24',
      fontStyle: 'bold',
      fontFamily: 'monospace'
    });
    this.timerText.setOrigin(0.5, 0.5);
    this.hudContainer.add(this.timerText);

    // === RIGHT SIDE: STATS ===
    const rightPadding = 24;
    const boxSpacing = 16;

    // Player count (far right)
    const statBoxWidth = 65;
    const playersX = screenWidth - rightPadding - (statBoxWidth / 2);
    const playersBg = this.add.rectangle(playersX, headerHeight / 2, statBoxWidth, 32, 0x1e293b, 1);
    playersBg.setStrokeStyle(1, 0x38bdf8);
    this.hudContainer.add(playersBg);

    const playersIcon = this.add.text(playersX - 16, headerHeight / 2, '👥', { fontSize: '16px' });
    playersIcon.setOrigin(0.5, 0.5);
    this.hudContainer.add(playersIcon);

    this.playerCountText = this.add.text(playersX + 14, headerHeight / 2, '1', {
      fontSize: '18px',
      color: '#38bdf8',
      fontStyle: 'bold',
      fontFamily: 'monospace'
    });
    this.playerCountText.setOrigin(0.5, 0.5);
    this.hudContainer.add(this.playerCountText);

    // Answers (next to player count)
    const answersX = playersX - statBoxWidth - boxSpacing;
    const answersBg = this.add.rectangle(answersX, headerHeight / 2, statBoxWidth, 32, 0x1e293b, 1);
    answersBg.setStrokeStyle(1, 0x4ade80);
    this.hudContainer.add(answersBg);

    const answersIcon = this.add.text(answersX - 16, headerHeight / 2, '✓', {
      fontSize: '18px',
      color: '#4ade80',
      fontStyle: 'bold'
    });
    answersIcon.setOrigin(0.5, 0.5);
    this.hudContainer.add(answersIcon);

    this.answersText = this.add.text(answersX + 14, headerHeight / 2, '0', {
      fontSize: '18px',
      color: '#4ade80',
      fontStyle: 'bold',
      fontFamily: 'monospace'
    });
    this.answersText.setOrigin(0.5, 0.5);
    this.hudContainer.add(this.answersText);

    // Kills (next to answers)
    const killsX = answersX - statBoxWidth - boxSpacing;
    const killsBg = this.add.rectangle(killsX, headerHeight / 2, statBoxWidth, 32, 0x1e293b, 1);
    killsBg.setStrokeStyle(1, 0xf87171);
    this.hudContainer.add(killsBg);

    const killsIcon = this.add.text(killsX - 16, headerHeight / 2, '💀', { fontSize: '16px' });
    killsIcon.setOrigin(0.5, 0.5);
    this.hudContainer.add(killsIcon);

    this.killsText = this.add.text(killsX + 14, headerHeight / 2, '0', {
      fontSize: '18px',
      color: '#f87171',
      fontStyle: 'bold',
      fontFamily: 'monospace'
    });
    this.killsText.setOrigin(0.5, 0.5);
    this.hudContainer.add(this.killsText);

    // Gun & Ammo (next to kills)
    const gunBoxWidth = 145;
    const gunX = killsX - (statBoxWidth / 2) - boxSpacing - (gunBoxWidth / 2);
    const gunBg = this.add.rectangle(gunX, headerHeight / 2, gunBoxWidth, 32, 0x1e293b, 1);
    gunBg.setStrokeStyle(1, 0xfde047);
    this.hudContainer.add(gunBg);

    const gunIcon = this.add.text(gunX - 58, headerHeight / 2, '🔫', { fontSize: '16px' });
    gunIcon.setOrigin(0.5, 0.5);
    this.hudContainer.add(gunIcon);

    this.gunText = this.add.text(gunX - 12, headerHeight / 2, 'Pistol', {
      fontSize: '16px',
      color: '#fde047',
      fontStyle: 'bold'
    });
    this.gunText.setOrigin(0.5, 0.5);
    this.hudContainer.add(this.gunText);

    this.ammoText = this.add.text(gunX + 48, headerHeight / 2, '6/6', {
      fontSize: '16px',
      color: '#fb923c',
      fontStyle: 'bold',
      fontFamily: 'monospace'
    });
    this.ammoText.setOrigin(0.5, 0.5);
    this.hudContainer.add(this.ammoText);

    // Set the HUD container to be fixed on screen
    this.hudContainer.setScrollFactor(0);
    this.hudContainer.setDepth(1000);

    // === WAITING TEXT (centered below header) ===
    const waitingY = headerHeight + 30;

    const waitingBg = this.add.rectangle(screenWidth / 2, waitingY, 380, 28, 0x78350f, 0.95);
    waitingBg.setStrokeStyle(1, 0xfbbf24);
    waitingBg.setScrollFactor(0);
    waitingBg.setDepth(1000);

    this.waitingText = this.add.text(screenWidth / 2, waitingY, '⚠ Waiting for game • Shooting disabled', {
      fontSize: '13px',
      color: '#fde047',
      fontStyle: 'bold',
      align: 'center'
    });
    this.waitingText.setOrigin(0.5, 0.5);
    this.waitingText.setScrollFactor(0);
    this.waitingText.setDepth(1001);
    this.waitingText.setVisible(true);

    // Pulsing animation
    this.tweens.add({
      targets: this.waitingText,
      alpha: 0.6,
      duration: 800,
      ease: 'Sine.easeInOut',
      yoyo: true,
      repeat: -1
    });

    (this as any).waitingBg = waitingBg;

    // === BOTTOM CENTER: CONTROLS HINT ===
    this.controlsBg = this.add.rectangle(screenWidth / 2, screenHeight - 25, 400, 32, 0x0f172a, 0.8);
    this.controlsBg.setStrokeStyle(1, 0x475569);
    this.controlsBg.setScrollFactor(0);
    this.controlsBg.setDepth(1000);

    this.controlsText = this.add.text(screenWidth / 2, screenHeight - 25, 'WASD Move  •  1-3 Weapons  •  Click/Space Shoot', {
      fontSize: '13px',
      color: '#e2e8f0',
      fontFamily: 'monospace'
    });
    this.controlsText.setOrigin(0.5, 0.5);
    this.controlsText.setScrollFactor(0);
    this.controlsText.setDepth(1001);

    // Handle window resize
    this.scale.on('resize', this.handleResize, this);
  }

  // Handle window resize to reposition elements
  private handleResize(gameSize: Phaser.Structs.Size): void {
    const width = gameSize.width;
    const height = gameSize.height;

    // Reposition controls bar at bottom center
    if (this.controlsBg) {
      this.controlsBg.setPosition(width / 2, height - 25);
    }
    if (this.controlsText) {
      this.controlsText.setPosition(width / 2, height - 25);
    }
  }

  // Format seconds to MM:SS
  private formatTime(seconds: number): string {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  }

  // Update timer display
  updateTimer(seconds: number): void {
    this.gameTimer = seconds;
    if (this.timerText) {
      this.timerText.setText(this.formatTime(seconds));
      
      if (seconds <= 60) {
        this.timerText.setColor('#ef4444');
        if (!this.timerActive) {
          this.timerActive = true;
          this.tweens.add({
            targets: this.timerText,
            alpha: 0.5,
            duration: 500,
            ease: 'Sine.easeInOut',
            yoyo: true,
            repeat: -1
          });
        }
      } else if (seconds <= 120) {
        this.timerText.setColor('#f97316');
      } else {
        this.timerText.setColor('#fbbf24');
      }
    }
  }

  // Update gun display
  updateGun(gunName: string): void {
    if (this.gunText) {
      this.gunText.setText(gunName);
    }
  }

  // Update ammo display
  updateAmmo(current: number, max: number): void {
    if (this.ammoText) {
      const text = current > 0 ? `${current}/${max}` : 'EMPTY';
      this.ammoText.setText(text);
      this.ammoText.setColor(current > 0 ? '#fb923c' : '#ef4444');
    }
  }

  // Update score display
  updateScore(kills: number, correctAnswers: number): void {
    if (this.killsText) {
      this.killsText.setText(kills.toString());
    }
    if (this.answersText) {
      this.answersText.setText(correctAnswers.toString());
    }
  }

  // Update player count
  updatePlayerCount(count: number): void {
    if (this.playerCountText) {
      this.playerCountText.setText(count.toString());
    }
  }

  // Show/hide waiting text
  setWaitingVisible(visible: boolean): void {
    if (this.waitingText) {
      this.waitingText.setVisible(visible);
    }
    if ((this as any).waitingBg) {
      (this as any).waitingBg.setVisible(visible);
    }
  }

  // Show game over screen
  showGameOver(kills: number, correctAnswers: number): void {
    const screenWidth = this.cameras.main.width;
    const screenHeight = this.cameras.main.height;

    // Hide waiting text if visible
    this.setWaitingVisible(false);

    // Create semi-transparent background
    const gameOverBg = this.add.rectangle(
      screenWidth / 2,
      screenHeight / 2,
      screenWidth,
      screenHeight,
      0x000000,
      0.7
    );
    gameOverBg.setScrollFactor(0);
    gameOverBg.setDepth(2000);

    // Create game over container
    const containerBg = this.add.rectangle(
      screenWidth / 2,
      screenHeight / 2,
      380,
      280,
      0x1e1e2e,
      0.95
    );
    containerBg.setStrokeStyle(3, 0xfbbf24);
    containerBg.setScrollFactor(0);
    containerBg.setDepth(2001);

    // Game over title
    const gameOverTitle = this.add.text(screenWidth / 2, screenHeight / 2 - 100, 'GAME OVER!', {
      fontSize: '42px',
      color: '#fbbf24',
      fontStyle: 'bold',
      fontFamily: 'Arial'
    });
    gameOverTitle.setOrigin(0.5, 0.5);
    gameOverTitle.setScrollFactor(0);
    gameOverTitle.setDepth(2002);

    // Stats - vertically stacked for clarity
    const statsStartY = screenHeight / 2 - 40;
    const statsSpacing = 35;

    // Kills stat row
    const killsRow = this.add.text(screenWidth / 2, statsStartY, `💀 Kills: ${kills}`, {
      fontSize: '22px',
      color: '#f87171',
      fontStyle: 'bold'
    });
    killsRow.setOrigin(0.5, 0.5);
    killsRow.setScrollFactor(0);
    killsRow.setDepth(2002);

    // Correct answers stat row
    const answersRow = this.add.text(screenWidth / 2, statsStartY + statsSpacing, `✓ Correct Answers: ${correctAnswers}`, {
      fontSize: '22px',
      color: '#4ade80',
      fontStyle: 'bold'
    });
    answersRow.setOrigin(0.5, 0.5);
    answersRow.setScrollFactor(0);
    answersRow.setDepth(2002);

    // Score calculation: kills * 100 + correctAnswers * 50
    const totalScore = kills * 100 + correctAnswers * 50;
    const scoreText = this.add.text(screenWidth / 2, statsStartY + statsSpacing * 2 + 10, `Total Score: ${totalScore}`, {
      fontSize: '28px',
      color: '#22d3ee',
      fontStyle: 'bold'
    });
    scoreText.setOrigin(0.5, 0.5);
    scoreText.setScrollFactor(0);
    scoreText.setDepth(2002);

    // Instruction text
    const instructionText = this.add.text(screenWidth / 2, screenHeight / 2 + 110, 'Check dashboard for final standings', {
      fontSize: '16px',
      color: '#94a3b8'
    });
    instructionText.setOrigin(0.5, 0.5);
    instructionText.setScrollFactor(0);
    instructionText.setDepth(2002);

    // Pulse animation on title
    this.tweens.add({
      targets: gameOverTitle,
      scaleX: 1.1,
      scaleY: 1.1,
      duration: 500,
      ease: 'Sine.easeInOut',
      yoyo: true,
      repeat: -1
    });
  }

}
