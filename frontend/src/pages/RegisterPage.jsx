import { Select } from "../components/FormControls.jsx";
import { Fragment } from "react";
import { truth, items, text } from "../lib/view.js";
import mountRegister from "../behaviors/register.js";
import { usePageBehaviors } from "../lib/behaviors.js";
import HeroSlideshow from "../components/HeroSlideshow.jsx";
export default function RegisterPage({ data }) {
  usePageBehaviors([mountRegister], data);
  return (
    <>
      <header className="register-header">
        <a href="/" aria-label="Agrilinks Nigeria home">
          <img
            src="/static/assets/logo/agrilinks-nigeria-logo.svg"
            alt="Agrilinks Nigeria"
          />
        </a>
        <a className="register-signin" href="/login">
          Log in <span aria-hidden="true">→</span>
        </a>
      </header>
      <main className="register-page">
        <section
          className="register-intro"
          aria-labelledby="register-intro-heading"
        >
          <div className="register-intro-copy">
            <p className="register-eyebrow">
              Your farm. Your harvest. Your next opportunity.
            </p>
            <h1 id="register-intro-heading">
              Grow more.
              <br />
              Reach more buyers.
            </h1>
            <p className="register-intro-description">
              Turn your hard work into new opportunities. Connect with buyers,
              discover fresh produce and livestock, and find practical support
              for your next harvest.
            </p>
            <ul className="register-highlights">
              <li>Buy and sell farm produce</li>
              <li>Learn and grow with other farmers</li>
              <li>Get weather insights and farming support</li>
            </ul>
            <a className="register-start" href="#register-form">
              Create your account <span aria-hidden="true">→</span>
            </a>
            <p className="register-invitation">
              For farmers, buyers, and everyone ready to grow.
            </p>
          </div>
          <div className="register-intro-photo">
            <HeroSlideshow imageId="register-photo" initialIndex={5} />
          </div>
        </section>
        <section
          className="auth-right register-form-section"
          aria-labelledby="register-heading"
        >
          <div className="auth-form-wrapper">
            <form
              id="register-form"
              action="/register"
              method="POST"
              encType="multipart/form-data"
            >
              <h2 id="register-heading"> Create Your Account </h2>
              <section
                className="form-section purpose-section"
                aria-labelledby="purpose-heading"
              >
                <h3 id="purpose-heading">
                  {" "}
                  How do you want to use Agrilinks Nigeria?{" "}
                </h3>
                <p className="field-help">
                  {" "}
                  Select all the activities that apply to you. You can change or
                  add to these choices later.{" "}
                </p>
                <div className="purpose-options">
                  <label className="purpose-option">
                    <input type="checkbox" name="purpose" defaultValue="sell" />
                    <span> 🌾 I want to sell agricultural products </span>
                  </label>
                  <label className="purpose-option">
                    <input type="checkbox" name="purpose" defaultValue="buy" />
                    <span> 🛒 I want to buy agricultural products </span>
                  </label>
                  <label className="purpose-option">
                    <input
                      type="checkbox"
                      name="purpose"
                      defaultValue="learn"
                    />
                    <span> 📚 I want to learn agriculture </span>
                  </label>
                </div>
              </section>
              <section
                className="form-section farm-type-section"
                id="farm-type-section"
                aria-labelledby="farm-type-heading"
              >
                <h3 id="farm-type-heading"> Your Agricultural Activities </h3>
                <p className="field-help">
                  {" "}
                  If you already practice agriculture, select the activities
                  that apply to you. If you are here to learn, you can leave
                  this section unselected and choose what you want to learn
                  later.{" "}
                </p>
                <div className="farm-options">
                  {truth(data?.FarmingTypes)
                    ? items(data?.FarmingTypes).map((item1, index1) => (
                        <Fragment key={index1}>
                          <label className="farm-option">
                            <input
                              type="checkbox"
                              name="farm_type"
                              defaultValue={text(item1?.Slug)}
                            />
                            <span>{text(item1?.Name)}</span>
                          </label>
                        </Fragment>
                      ))
                    : null}
                </div>
              </section>
              <section
                className="form-section personal-information"
                aria-labelledby="personal-information-heading"
              >
                <h3 id="personal-information-heading">
                  {" "}
                  Personal Information{" "}
                </h3>
                <div className="row">
                  <div className="form-group">
                    <label htmlFor="first-name"> First Name </label>
                    <input
                      name="first-name"
                      id="first-name"
                      type="text"
                      placeholder="Enter your first name"
                      autoComplete="given-name"
                      maxLength="100"
                      required={true}
                    />
                  </div>
                  <div className="form-group">
                    <label htmlFor="last-name"> Last Name </label>
                    <input
                      name="last-name"
                      id="last-name"
                      type="text"
                      placeholder="Enter your last name"
                      autoComplete="family-name"
                      maxLength="100"
                      required={true}
                    />
                  </div>
                </div>
                <div className="form-group">
                  <label htmlFor="email"> Email Address </label>
                  <input
                    name="email"
                    id="email"
                    type="email"
                    placeholder="Enter your email"
                    autoComplete="email"
                    maxLength="255"
                    required={true}
                  />
                </div>
                <div className="form-group">
                  <label htmlFor="phone"> Phone Number </label>
                  <input
                    name="phone"
                    id="phone"
                    type="tel"
                    inputMode="tel"
                    placeholder="Enter phone number"
                    autoComplete="tel"
                    maxLength="20"
                    required={true}
                  />
                </div>
              </section>
              <section
                className="form-section location-section"
                aria-labelledby="location-heading"
              >
                <h3 id="location-heading"> Where are you located? </h3>
                <p className="field-help">
                  {" "}
                  Your location helps Agrilinks Nigeria connect you with
                  relevant agricultural opportunities, markets, services,
                  learning spaces, and people near you.{" "}
                </p>
                <div className="form-group">
                  <label htmlFor="country"> Country </label>
                  <Select
                    name="country"
                    id="country"
                    autoComplete="country-name"
                    required={true}
                  >
                    <option value=""> Select your country </option>
                    <option value="Nigeria" selected={true}>
                      {" "}
                      Nigeria{" "}
                    </option>
                  </Select>
                </div>
                <div className="form-group">
                  <label htmlFor="state"> State </label>
                  <Select
                    name="state"
                    id="state"
                    autoComplete="address-level1"
                    required={true}
                  >
                    <option value=""> Select your state </option>
                  </Select>
                </div>
                <div className="form-group">
                  <label htmlFor="lga"> Local Government Area (LGA) </label>
                  <Select
                    name="lga"
                    id="lga"
                    autoComplete="address-level2"
                    required={true}
                    disabled={true}
                  >
                    <option value=""> Select your LGA </option>
                  </Select>
                </div>
                <div className="form-group">
                  <label htmlFor="community"> Community / Town </label>
                  <input
                    name="community"
                    id="community"
                    type="text"
                    placeholder="Enter your community or town"
                    autoComplete="address-level3"
                    maxLength="150"
                    required={true}
                  />
                </div>
              </section>
              <section
                className="form-section security-section"
                aria-labelledby="security-heading"
              >
                <h3 id="security-heading"> Secure Your Account </h3>
                <div className="form-group">
                  <label htmlFor="password"> Password </label>
                  <div className="password-wrapper">
                    <input
                      id="password"
                      name="password"
                      type="password"
                      placeholder="Create a password"
                      autoComplete="new-password"
                      minLength="8"
                      maxLength="128"
                      required={true}
                    />
                    <button
                      type="button"
                      className="toggle-password"
                      id="toggle-password"
                      aria-label="Show password"
                      aria-controls="password"
                      aria-pressed="false"
                    >
                      <i className="fa-solid fa-eye" aria-hidden="true"></i>
                    </button>
                  </div>
                  <p className="field-help">
                    {" "}
                    Password must be at least 8 characters.{" "}
                  </p>
                </div>
                <div className="form-group">
                  <label htmlFor="confirm-password"> Confirm Password </label>
                  <div className="password-wrapper">
                    <input
                      id="confirm-password"
                      name="confirm-password"
                      type="password"
                      placeholder="Confirm your password"
                      autoComplete="new-password"
                      minLength="8"
                      maxLength="128"
                      required={true}
                    />
                    <button
                      type="button"
                      className="toggle-password"
                      id="toggle-confirm-password"
                      aria-label="Show confirm password"
                      aria-controls="confirm-password"
                      aria-pressed="false"
                    >
                      <i className="fa-solid fa-eye" aria-hidden="true"></i>
                    </button>
                  </div>
                </div>
              </section>
              <button type="submit" id="register-submit">
                {" "}
                Continue to selfie{" "}
              </button>
              <section
                className="form-section profile-section"
                aria-labelledby="profile-heading"
                hidden={true}
              >
                <h3 id="profile-heading">
                  {" "}
                  Take a selfie and confirm your phone{" "}
                </h3>
                <p className="field-help">
                  {" "}
                  Take a clear selfie for private account verification. It will
                  not become your public profile picture. Allow camera access
                  when your device asks.{" "}
                </p>
                <div className="form-group selfie-camera">
                  <div
                    id="selfie-viewport"
                    className="selfie-viewport"
                    hidden={true}
                  >
                    <video
                      id="selfie-video"
                      aria-label="Live selfie camera preview"
                      autoPlay={true}
                      muted={true}
                      playsInline={true}
                      hidden={true}
                    ></video>
                    <img
                      id="selfie-preview"
                      alt="Your captured selfie"
                      hidden={true}
                    />
                  </div>
                  <button
                    type="button"
                    id="open-selfie-camera"
                    className="registration-back-button"
                    aria-describedby="selfie-status"
                  >
                    {" "}
                    Open camera{" "}
                  </button>
                  <button
                    type="button"
                    id="take-selfie"
                    className="registration-back-button selfie-capture-button"
                    disabled={true}
                    hidden={true}
                  >
                    {" "}
                    Take selfie{" "}
                  </button>
                  <button
                    type="button"
                    id="retake-selfie"
                    className="registration-back-button"
                    hidden={true}
                  >
                    {" "}
                    Retake selfie{" "}
                  </button>
                  <p
                    id="selfie-status"
                    className="field-help"
                    role="status"
                    aria-live="polite"
                  >
                    {" "}
                    Open your camera to take a selfie.{" "}
                  </p>
                </div>
                <div className="form-group phone-confirmation">
                  <h3>Confirm your phone number</h3>
                  <p className="field-help">
                    We’ll send a code to{" "}
                    <strong id="confirmation-phone"></strong>. Verify it, then
                    create your transaction PIN.
                  </p>
                  <button
                    type="button"
                    id="send-registration-code"
                    className="registration-back-button"
                  >
                    Send confirmation code
                  </button>
                  <p
                    id="confirmation-status"
                    className="field-help"
                    role="status"
                  ></p>
                  <label htmlFor="registration-code">
                    SMS confirmation code
                  </label>
                  <input
                    id="registration-code"
                    name="registration_code"
                    type="text"
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    pattern="[0-9]{6}"
                    maxLength="6"
                    placeholder="6-digit code"
                    disabled={true}
                  />
                  <button
                    type="button"
                    id="verify-registration-phone"
                    className="registration-back-button"
                    disabled={true}
                  >
                    Verify phone number
                  </button>
                </div>
                <div
                  id="registration-pin-section"
                  className="form-group"
                  hidden={true}
                >
                  <h3>Create your transaction PIN</h3>
                  <p id="registration-pin-help" className="field-help">
                    Choose six digits to authorize wallet payments and
                    withdrawals. Keep your PIN safe and never reveal it to
                    anyone. Agrilinks Nigeria staff will never ask for your PIN.
                  </p>
                  <label htmlFor="transaction-pin">Transaction PIN</label>
                  <input
                    id="transaction-pin"
                    name="transaction_pin"
                    type="password"
                    inputMode="numeric"
                    autoComplete="new-password"
                    pattern="[0-9]{6}"
                    minLength="6"
                    maxLength="6"
                    aria-describedby="registration-pin-help"
                    disabled={true}
                  />
                  <label htmlFor="transaction-pin-confirmation">
                    Confirm transaction PIN
                  </label>
                  <input
                    id="transaction-pin-confirmation"
                    name="transaction_pin_confirmation"
                    type="password"
                    inputMode="numeric"
                    autoComplete="new-password"
                    pattern="[0-9]{6}"
                    minLength="6"
                    maxLength="6"
                    disabled={true}
                  />
                  <p
                    id="registration-pin-status"
                    className="field-help"
                    role="status"
                  ></p>
                </div>
                <button
                  type="button"
                  id="back-to-details"
                  className="registration-back-button"
                >
                  {" "}
                  Back to account details{" "}
                </button>
                <button
                  type="submit"
                  id="complete-registration"
                  disabled={true}
                >
                  {" "}
                  Complete Registration{" "}
                </button>
              </section>
              <p className="auth-footer">
                {" "}
                Already have an account? <a href="/login"> Login </a>
              </p>
            </form>
          </div>
        </section>
      </main>
    </>
  );
}
