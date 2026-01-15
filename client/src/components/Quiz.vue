<template>
  <div v-if="showQuiz" class="quiz-overlay">
    <div class="quiz-container">
      <h2 class="quiz-title">{{ quizTitle }}</h2>
      <p class="quiz-subtitle">Answer 3 questions to {{ quizAction }}</p>

      <div class="quiz-content">
        <div class="question-counter">
          Question {{ currentQuestionIndex + 1 }} of {{ totalQuestions }}
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
            {{ option }}
          </button>
        </div>

        <div v-if="showFeedback" class="feedback">
          <p v-if="isCorrect" class="correct-feedback">Correct!</p>
          <p v-else class="incorrect-feedback">
            Incorrect! The answer was: {{ currentQuestion.options[currentQuestion.correct_answer] }}
          </p>
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

  // Select 3 random questions
  const shuffled = [...allQuestions.value].sort(() => Math.random() - 0.5);
  questions.value = shuffled.slice(0, totalQuestions);

  showQuiz.value = true;
};

const selectAnswer = (key: string) => {
  selectedAnswer.value = key;
  isCorrect.value = key === String(currentQuestion.value.correct_answer);

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
.quiz-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background: rgba(0, 0, 0, 0.9);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
}

.quiz-container {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 2rem;
  border-radius: 20px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  max-width: 500px;
  width: 90%;
}

.quiz-title {
  color: white;
  font-size: 2rem;
  margin-bottom: 0.5rem;
  text-align: center;
  text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.3);
}

.quiz-subtitle {
  color: rgba(255, 255, 255, 0.9);
  text-align: center;
  margin-bottom: 1.5rem;
}

.quiz-content {
  background: white;
  padding: 1.5rem;
  border-radius: 10px;
}

.question-counter {
  text-align: center;
  color: #666;
  margin-bottom: 1rem;
  font-size: 0.9rem;
}

.question {
  font-size: 1.2rem;
  margin-bottom: 1.5rem;
  color: #333;
  text-align: center;
  font-weight: 600;
}

.options {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.75rem;
}

.option-button {
  padding: 1rem;
  border: 2px solid #e0e0e0;
  background: white;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.3s;
  font-size: 1rem;
  font-weight: 500;
}

.option-button:hover:not(:disabled) {
  background: #f5f5f5;
  border-color: #667eea;
  transform: translateY(-2px);
}

.option-button:disabled {
  cursor: not-allowed;
}

.option-button.correct {
  background: #4caf50;
  color: white;
  border-color: #4caf50;
}

.option-button.incorrect {
  background: #f44336;
  color: white;
  border-color: #f44336;
}

.feedback {
  margin-top: 1rem;
  padding: 0.75rem;
  border-radius: 8px;
  text-align: center;
}

.correct-feedback {
  color: #4caf50;
  font-weight: bold;
}

.incorrect-feedback {
  color: #f44336;
}
</style>