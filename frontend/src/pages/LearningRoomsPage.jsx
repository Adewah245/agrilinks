import { Select, Textarea } from "../components/FormControls.jsx";
import { Fragment } from "react";
import { truth, items, text } from "../lib/view.js";
export default function LearningRoomsPage({ data }) {
  return (
    <>
      <nav className="learning-nav">
        <a href="/learn">← Your learning space</a>
        <a href="/chat">Messages</a>
      </nav>
      <main className="learning-main">
        <header>
          <h1>Learn together</h1>
          <p>
            Join a room for your farming activity, or start one and welcome
            other learners.
          </p>
        </header>
        <div className="learning-columns">
          <section className="learning-card">
            <h2>Find a room</h2>
            <form method="get" action="/learn/rooms">
              <label htmlFor="room-filter">Farming activity</label>
              <Select id="room-filter" name="farming_type_id">
                <option value="0">All farming activities</option>
                {truth(data?.FarmingTypes)
                  ? items(data?.FarmingTypes).map((item1, index1) => (
                      <Fragment key={index1}>
                        <option
                          value={text(item1?.ID)}
                          selected={truth(data?.SelectedType === item1?.ID)}
                        >
                          {text(item1?.Name)}
                        </option>
                      </Fragment>
                    ))
                  : null}
              </Select>
              <button type="submit">Find rooms</button>
            </form>
            {truth(data?.Rooms) ? (
              items(data?.Rooms).map((item2, index2) => (
                <Fragment key={index2}>
                  <article className="journey-summary">
                    <h3>{text(item2?.Title)}</h3>
                    <p>
                      {text(item2?.FarmingType)} · {text(item2?.MemberCount)}{" "}
                      members
                    </p>
                    <p>{text(item2?.Description)}</p>
                    {truth(item2?.Joined) ? (
                      <>
                        <a
                          href={"/chat/view?id=" + text(item2?.ConversationID)}
                        >
                          Open room →
                        </a>
                      </>
                    ) : (
                      <>
                        <form method="post" action="/learn/rooms/join">
                          <input
                            type="hidden"
                            name="room_id"
                            defaultValue={text(item2?.ConversationID)}
                          />
                          <button type="submit">Join room</button>
                        </form>
                      </>
                    )}
                  </article>
                </Fragment>
              ))
            ) : (
              <>
                <p>No rooms were found. You can start one below.</p>
              </>
            )}
          </section>
          <section className="learning-card">
            <h2>Start a room</h2>
            <p>
              Other people using Agrilinks Nigeria will be able to find and join
              this room.
            </p>
            <form method="post" action="/learn/rooms/create">
              <label htmlFor="room-type">Farming activity</label>
              <Select id="room-type" name="farming_type_id" required={true}>
                <option value="">Choose an activity</option>
                {truth(data?.FarmingTypes)
                  ? items(data?.FarmingTypes).map((item3, index3) => (
                      <Fragment key={index3}>
                        <option value={text(item3?.ID)}>
                          {text(item3?.Name)}
                        </option>
                      </Fragment>
                    ))
                  : null}
              </Select>
              <label htmlFor="room-title">Room name</label>
              <input
                id="room-title"
                name="title"
                maxLength="100"
                required={true}
              />
              <label htmlFor="room-description">
                What will you learn together?
              </label>
              <Textarea
                id="room-description"
                name="description"
                rows="4"
                maxLength="2000"
              ></Textarea>
              <button type="submit">Create room</button>
            </form>
          </section>
        </div>
      </main>
    </>
  );
}
