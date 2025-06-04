import Controller from '@ember/controller';
import { service } from '@ember/service';

export default class extends Controller {
  @service fastboot;

  // TODO: Add shoebox support?

  get debug() {
    return JSON.stringify(
      {
        isFastBoot: this.fastboot.isFastBoot,
        request: this.fastboot.request,
        requestHost: this.fastboot.request?.host,
        response: this.fastboot.response,
        metadata: this.fastboot.metadata,
      },
      null,
      2,
    );
  }
}
