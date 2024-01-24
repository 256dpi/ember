import Route from '@ember/routing/route';

export default class extends Route {
  model() {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({ message: 'Hello world!' });
      }, 500);
    });
  }
}
