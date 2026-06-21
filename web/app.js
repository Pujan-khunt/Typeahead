const suggestion_list = document.getElementById("suggestion-list");
const search_bar = document.getElementById("search-bar-input");

let currentController = null;

search_bar.addEventListener("input", loadSuggestions);
search_bar.addEventListener("keydown", handleSearchInput)
search_bar.value = "";
suggestion_list.replaceChildren();

async function loadSuggestions(input_event) {
  // Retrieve partial query from user.
  const prefix = input_event.target.value;

  // Terminate requests early.
  if (prefix === "" || prefix.length > 50) {
    suggestion_list.classList.remove("visible");
    suggestion_list.replaceChildren();
    return;
  }

  // Abort previous fetch request.
  if (currentController) {
    currentController.abort();
  }
  currentController = new AbortController();

  try {
    // Fetch new suggestions from backend.
    const suggestions = await fetchSuggestions(prefix, 10, currentController.signal);

    // Remove existing suggestions.
    suggestion_list.replaceChildren();

    // Display all the suggestions.
    for (const suggestion of suggestions) {
      if (suggestion.length < 50) {
        const list_item = document.createElement('li');
        list_item.innerHTML = suggestion.substring(0, prefix.length) + "<b>" + suggestion.substring(prefix.length) + "</b>";
        suggestion_list.appendChild(list_item);
      }
    }
    suggestion_list.classList.add("visible");
  } catch (error) {
    if (error.name !== "AbortError") {
      console.error("Failed to fetch suggestions:", error);
    }
  }
}

async function fetchSuggestions(prefix, limit, signal) {
  const url = new URL("http://localhost:8080/api/v1/suggestions")
  url.searchParams.append("prefix", prefix)
  url.searchParams.append("limit", limit)

  const response = await fetch(url, { signal });
  const suggestions = await response.json();
  return suggestions;
}

async function handleSearchInput(event) {
  if (event.key !== "Enter") {
    return;
  }

  const query = search_bar.value;
  if (query === "") {
    return
  }

  await fetch("http://127.0.0.1:8080/api/v1/search", {
    method: "POST",
    body: query
  });

  search_bar.value = '';
  suggestion_list.replaceChildren();
}
