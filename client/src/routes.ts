import HomePage from "./views/HomePage.vue"
import DashboardPage from "./views/DashboardPage.vue";
import GamePage from "./views/GamePage.vue";

export default [
   { path: '/', component: HomePage },
   { path: '/dashboard/:code', component: DashboardPage },
   { path: '/game/:code', component: GamePage }
];