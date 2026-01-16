<template>
  <div v-if="showQuiz" class="quiz-overlay">
    <div class="quiz-container">
      <div class="quiz-header">
        <div class="quiz-icon" :class="currentQuizType === 'death' ? 'icon-death' : 'icon-ammo'">
          <svg v-if="currentQuizType === 'death'" class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <svg v-else class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        </div>
        <h2 class="quiz-title">{{ quizTitle }}</h2>
        <p class="quiz-subtitle">Answer 3 questions to {{ quizAction }}</p>
      </div>

      <div class="quiz-content">
        <div class="question-counter">
          <span class="counter-current">{{ currentQuestionIndex + 1 }}</span>
          <span class="counter-divider">/</span>
          <span class="counter-total">{{ totalQuestions }}</span>
        </div>

        <div class="question">
          {{ currentQuestion.question }}
        </div>

        <div class="options">
          <button
            v-for="(option, key) in currentQuestion.options"
            :key="key"
            @click="selectAnswer(key)"
            :class="['option-button', {
              'correct': showFeedback && key === String(currentQuestion.correct_answer),
              'incorrect': showFeedback && key === selectedAnswer && key !== String(currentQuestion.correct_answer)
            }]"
            :disabled="showFeedback"
          >
            <span class="option-key">{{ key }}</span>
            <span class="option-text">{{ option }}</span>
          </button>
        </div>

        <div v-if="showFeedback" class="feedback" :class="isCorrect ? 'feedback-correct' : 'feedback-incorrect'">
          <svg v-if="isCorrect" class="feedback-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          <svg v-else class="feedback-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
          <p v-if="isCorrect">Correct!</p>
          <p v-else>
            The answer was: {{ currentQuestion.options[currentQuestion.correct_answer] }}
          </p>
        </div>

        <!-- Progress bar -->
        <div class="progress-bar">
          <div 
            class="progress-fill" 
            :style="{ width: ((currentQuestionIndex + (showFeedback ? 1 : 0)) / totalQuestions) * 100 + '%' }"
          ></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';

interface Question {
  question_number: number;
  question: string;
  options: { [key: string]: string | number };
  correct_answer: number;
}

type QuizType = 'death' | 'ammo';

const emit = defineEmits(['quizCompleted']);

const showQuiz = ref(false);
const questions = ref<Question[]>([]);
const currentQuestionIndex = ref(0);
const selectedAnswer = ref<string | null>(null);
const showFeedback = ref(false);
const isCorrect = ref(false);
const allQuestions = ref<Question[]>([]);
const totalQuestions = 3;
const currentQuizType = ref<QuizType>('ammo');
const correctAnswersCount = ref(0); // Track correct answers in this quiz session

const currentQuestion = computed(() => questions.value[currentQuestionIndex.value]);
const quizTitle = computed(() => currentQuizType.value === 'death' ? 'You Died!' : 'Out of Ammo!');
const quizAction = computed(() => currentQuizType.value === 'death' ? 'respawn' : 'reload');

const loadQuestions = async () => {
  try {
    const response = await fetch('/data/questions.json');
    const data = await response.json();
    allQuestions.value = data;
  } catch (error) {
    console.error('Failed to load questions:', error);
  }
};

const startQuiz = (type: QuizType = 'ammo') => {
  // Reset quiz state
  currentQuizType.value = type;
  currentQuestionIndex.value = 0;
  selectedAnswer.value = null;
  showFeedback.value = false;
  correctAnswersCount.value = 0; // Reset correct answers count

  // Select 3 random questions
  const shuffled = [...allQuestions.value].sort(() => Math.random() - 0.5);
  questions.value = shuffled.slice(0, totalQuestions);

  showQuiz.value = true;
};

const selectAnswer = (key: string) => {
  selectedAnswer.value = key;
  isCorrect.value = key === String(currentQuestion.value.correct_answer);

  // Track correct answers
  if (isCorrect.value) {
    correctAnswersCount.value++;
  }

  showFeedback.value = true;

  // Auto proceed after 1.5 seconds
  setTimeout(() => {
    if (currentQuestionIndex.value < questions.value.length - 1) {
      nextQuestion();
    } else {
      completeQuiz();
    }
  }, 1500);
};

const nextQuestion = () => {
  currentQuestionIndex.value++;
  selectedAnswer.value = null;
  showFeedback.value = false;
  isCorrect.value = false;
};

const completeQuiz = () => {
  // Always complete successfully (no need to answer correctly)
  emit('quizCompleted', true);
  showQuiz.value = false;

  // Get the game scene and handle completion
  const game = (window as any).gameManager?.getGame();
  if (game) {
    const mainScene = game.scene.getScene('MainScene');
    if (mainScene) {
      // Send correct answers to server for score calculation
      mainScene.sendCorrectAnswers(correctAnswersCount.value);

      if (currentQuizType.value === 'death') {
        // Handle respawn
        if ((window as any).onQuizComplete) {
          (window as any).onQuizComplete();
        }
      } else {
        // Handle ammo reload
        mainScene.reloadAmmo();
      }
    }
  }
};

// Load questions on mount
onMounted(() => {
  loadQuestions();
});

// Expose methods for external calls
defineExpose({
  startQuiz
});
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700;800&display=swap');

.quiz-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background: rgba(0, 0, 0, 0.85);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  font-family: 'Outfit', sans-serif;
}

.quiz-container {
  background: linear-gradient(145deg, rgba(13, 17, 23, 0.98) 0%, rgba(13, 17, 23, 0.99) 100%);
  border: 1px solid rgba(123, 184, 204, 0.3);
  padding: 2rem;
  border-radius: 1.5rem;
  box-shadow: 
    0 25px 60px rgba(0, 0, 0, 0.6),
    0 0 40px rgba(123, 184, 204, 0.1);
  max-width: 520px;
  width: 90%;
}

.quiz-header {
  text-align: center;
  margin-bottom: 1.5rem;
}

.quiz-icon {
  width: 4rem;
  height: 4rem;
  border-radius: 1rem;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1rem;
}

.quiz-icon.icon-death {
  background: linear-gradient(135deg, #FF7A7A 0%, #d94545 100%);
  color: white;
  box-shadow: 0 8px 25px rgba(255, 122, 122, 0.4);
}

.quiz-icon.icon-ammo {
  background: linear-gradient(135deg, #FFD980 0%, #FFBE7A 100%);
  color: #0d1117;
  box-shadow: 0 8px 25px rgba(255, 217, 128, 0.4);
}

.quiz-title {
  color: white;
  font-size: 2rem;
  font-weight: 800;
  margin-bottom: 0.5rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.quiz-subtitle {
  color: rgba(255, 255, 255, 0.8);
  font-size: 1.1rem;
  font-weight: 500;
}

.quiz-content {
  background: linear-gradient(145deg, #f5f0e6 0%, #ebe5d9 100%);
  padding: 1.5rem;
  border-radius: 1rem;
  border: 1px solid rgba(200, 180, 150, 0.5);
  box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.05);
}

.question-counter {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.25rem;
  margin-bottom: 1rem;
  font-size: 0.9rem;
}

.counter-current {
  color: #5A9CB5;
  font-weight: 800;
  font-size: 1.3rem;
}

.counter-divider {
  color: rgba(0, 0, 0, 0.4);
}

.counter-total {
  color: rgba(0, 0, 0, 0.5);
}

.question {
  font-size: 1.25rem;
  margin-bottom: 1.5rem;
  color: #2c3e50;
  text-align: center;
  font-weight: 700;
  line-height: 1.5;
}

.options {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.75rem;
}

.option-button {
  padding: 1rem;
  background: white;
  border: 2px solid #d4cfc5;
  border-radius: 0.75rem;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  text-align: left;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.option-key {
  width: 1.75rem;
  height: 1.75rem;
  background: #5A9CB5;
  border-radius: 0.375rem;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  color: white;
  font-size: 0.85rem;
  flex-shrink: 0;
}

.option-text {
  color: #2c3e50;
  font-weight: 600;
  font-size: 1rem;
}

.option-button:hover:not(:disabled) {
  background: #f8f6f2;
  border-color: #5A9CB5;
  transform: translateY(-2px);
  box-shadow: 0 8px 20px rgba(90, 156, 181, 0.2);
}

.option-button:hover:not(:disabled) .option-key {
  background: #4a8a9f;
}

.option-button:hover:not(:disabled) .option-text {
  color: #5A9CB5;
}

.option-button:disabled {
  cursor: not-allowed;
}

.option-button.correct {
  background: #e8f5e9;
  border-color: #4caf50;
}

.option-button.correct .option-key {
  background: #4caf50;
  color: white;
}

.option-button.correct .option-text {
  color: #2e7d32;
}

.option-button.incorrect {
  background: #ffebee;
  border-color: #ef5350;
}

.option-button.incorrect .option-key {
  background: #ef5350;
  color: white;
}

.option-button.incorrect .option-text {
  color: #c62828;
}

.feedback {
  margin-top: 1rem;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
}

.feedback-icon {
  width: 1.25rem;
  height: 1.25rem;
  flex-shrink: 0;
}

.feedback-correct {
  background: #e8f5e9;
  border: 1px solid #a5d6a7;
  color: #2e7d32;
}

.feedback-incorrect {
  background: #ffebee;
  border: 1px solid #ef9a9a;
  color: #c62828;
}

.progress-bar {
  margin-top: 1.25rem;
  height: 6px;
  background: #d4cfc5;
  border-radius: 3px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #5A9CB5 0%, #7BB8CC 100%);
  border-radius: 3px;
  transition: width 0.3s ease;
}
</style>
